# Terraform Provider for Hook Service

A Terraform provider for managing groups and access in the [Canonical Hook Service](https://github.com/canonical/hook-service-operator) (part of the Canonical Identity Platform).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (to build the provider)
- A running Hook Service instance

## Provider Configuration

```hcl
terraform {
  required_providers {
    hookservice = {
      source  = "canonical/hookservice"
      version = "~> 0.1"
    }
  }
}

provider "hookservice" {
  host  = "http://<hook-service-ip>:8000"
  token = var.hook_service_token  # optional, for authenticated APIs
}
```

### Configuration Reference

| Argument | Description | Required | Environment Variable |
|----------|-------------|----------|---------------------|
| `host` | The Hook Service API URL (e.g. `http://10.0.0.1:8000`) | Yes | `HOOK_SERVICE_HOST` |
| `token` | Bearer token for API authentication | No | `HOOK_SERVICE_TOKEN` |

### Obtaining a Token

If you have the `oauth` relation set up with your Hook Service charm, obtain a token with:

```shell
export HOOK_SERVICE_TOKEN=$(juju run hook-service/0 get-access-token --format=json | jq -r '.["hook-service/0"].results.token')
```

## Resources

### `hookservice_group`

Manages a group in the Hook Service.

#### Example

```hcl
resource "hookservice_group" "engineering" {
  name        = "engineering"
  description = "Engineering team"
  type        = "local"
}
```

#### Argument Reference

| Argument | Description | Required | Default |
|----------|-------------|----------|---------|
| `name` | The name of the group | Yes | - |
| `description` | A description of the group | No | `""` |
| `type` | The group type (e.g. `local`) | No | `"local"` |

#### Attribute Reference

| Attribute | Description |
|-----------|-------------|
| `id` | The unique identifier of the group (assigned by the API) |

---

### `hookservice_group_users`

Manages the set of users belonging to a group. This resource tracks only the users you declare; users added to the group outside Terraform are not affected.

#### Example

```hcl
resource "hookservice_group_users" "engineering_members" {
  group_id = hookservice_group.engineering.id
  emails = [
    "alice@example.com",
    "bob@example.com",
    "charlie@example.com",
  ]
}
```

#### Argument Reference

| Argument | Description | Required |
|----------|-------------|----------|
| `group_id` | The ID of the group | Yes |
| `emails` | Set of user email addresses to add to the group | Yes |

---

### `hookservice_group_app`

Grants a group access to an application (identified by its OAuth client ID).

#### Example

```hcl
resource "hookservice_group_app" "engineering_app" {
  group_id  = hookservice_group.engineering.id
  client_id = "my-oauth-client-id"
}
```

#### Argument Reference

| Argument | Description | Required |
|----------|-------------|----------|
| `group_id` | The ID of the group | Yes |
| `client_id` | The OAuth client ID of the application | Yes |

---

## Data Sources

### `hookservice_groups`

Reads all groups from the Hook Service.

#### Example

```hcl
data "hookservice_groups" "all" {}

output "all_group_names" {
  value = [for g in data.hookservice_groups.all.groups : g.name]
}
```

#### Attribute Reference

| Attribute | Description |
|-----------|-------------|
| `groups` | List of group objects, each with `id`, `name`, `description`, and `type` |

---

## Full Example

This example creates a group, adds users, and grants the group access to an application:

```hcl
terraform {
  required_providers {
    hookservice = {
      source = "canonical/hookservice"
    }
  }
}

provider "hookservice" {
  host  = "http://10.0.0.1:8000"
  token = var.hook_service_token
}

variable "hook_service_token" {
  type      = string
  sensitive = true
}

variable "client_id" {
  type        = string
  description = "The OAuth client ID of the application"
}

# Create a group for the engineering team
resource "hookservice_group" "engineering" {
  name        = "engineering"
  description = "Engineering team access group"
  type        = "local"
}

# Add team members
resource "hookservice_group_users" "engineering_members" {
  group_id = hookservice_group.engineering.id
  emails = [
    "alice@example.com",
    "bob@example.com",
  ]
}

# Grant the group access to the application
resource "hookservice_group_app" "engineering_app" {
  group_id  = hookservice_group.engineering.id
  client_id = var.client_id
}

# You can also read existing groups
data "hookservice_groups" "all" {}

output "engineering_group_id" {
  value = hookservice_group.engineering.id
}
```

Apply the configuration:

```shell
# Set the required environment variables
export HOOK_SERVICE_HOST="http://<hook-service-ip>:8000"
export HOOK_SERVICE_TOKEN="<your-bearer-token>"

# Initialize and apply
terraform init
terraform plan -var="client_id=my-app-client-id" -var="hook_service_token=$HOOK_SERVICE_TOKEN"
terraform apply -var="client_id=my-app-client-id" -var="hook_service_token=$HOOK_SERVICE_TOKEN"
```

Or using a `terraform.tfvars` file:

```hcl
# terraform.tfvars
hook_service_token = "your-token-here"
client_id          = "my-app-client-id"
```

```shell
terraform init
terraform apply
```

## Multiple Groups with Shared App Access

```hcl
locals {
  teams = {
    engineering = {
      description = "Engineering team"
      members     = ["alice@example.com", "bob@example.com"]
    }
    design = {
      description = "Design team"
      members     = ["carol@example.com", "dave@example.com"]
    }
    qa = {
      description = "QA team"
      members     = ["eve@example.com"]
    }
  }
}

resource "hookservice_group" "teams" {
  for_each    = local.teams
  name        = each.key
  description = each.value.description
  type        = "local"
}

resource "hookservice_group_users" "members" {
  for_each = local.teams
  group_id = hookservice_group.teams[each.key].id
  emails   = toset(each.value.members)
}

resource "hookservice_group_app" "app_access" {
  for_each  = local.teams
  group_id  = hookservice_group.teams[each.key].id
  client_id = var.client_id
}
```

## Development

### Building

```shell
go build -o terraform-provider-hookservice
```

### Testing

```shell
go test ./...
```
