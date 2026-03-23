terraform {
  required_providers {
    hookservice = {
      source = "canonical/hookservice"
    }
  }
}

# Configure the provider.
# The host and token can also be set via environment variables:
#   HOOK_SERVICE_HOST and HOOK_SERVICE_TOKEN
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

# Create a group
resource "hookservice_group" "engineering" {
  name        = "engineering"
  description = "Engineering team"
  type        = "local"
}

# Add users to the group
resource "hookservice_group_users" "engineering_members" {
  group_id = hookservice_group.engineering.id
  emails = [
    "alice@example.com",
    "bob@example.com",
  ]
}

# Grant the group access to an application
resource "hookservice_group_app" "engineering_app" {
  group_id  = hookservice_group.engineering.id
  client_id = var.client_id
}

# Read all existing groups
data "hookservice_groups" "all" {}

output "all_groups" {
  value = data.hookservice_groups.all.groups
}

output "engineering_group_id" {
  value = hookservice_group.engineering.id
}
