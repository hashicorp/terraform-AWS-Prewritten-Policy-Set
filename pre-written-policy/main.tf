// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

locals {
  policy_set_name        = "${var.name}-policy-set"
  policy_set_description = "Policy set created via terraform to evaluate resources against Sentinel policies"
  policy_set_kind        = "sentinel"
  sentinel_version       = "0.26.0"

  unzipped_policy_dir    = data.external.github_release.result["unzip_dir"]
}

data "external" "github_release" {
  program = ["python3", "${path.module}/fetch_python.py"]
  query = {
    action = "download"
    name   = var.name
    owner  = var.policy_owner
    repo   = var.policy_github_repository
    run_id = timestamp()
  }
}

# ------------------------------------------------  
# Policy Set creation
# ------------------------------------------------  

data "tfe_slug" "this" {
  depends_on = [data.external.github_release]
  source_path = local.unzipped_policy_dir
}

data "tfe_workspace_ids" "workspaces" {
  names        = var.policy_set_workspace_names
  organization = var.tfe_organization
}

resource "tfe_policy_set" "workspace_scoped_policy_set" {
  depends_on = [data.tfe_slug.this]

  name                = local.policy_set_name
  description         = local.policy_set_description
  organization        = var.tfe_organization
  kind                = local.policy_set_kind
  policy_tool_version = local.sentinel_version
  agent_enabled       = true
  workspace_ids       = values(data.tfe_workspace_ids.workspaces.ids)

  slug = data.tfe_slug.this
}

# ------------------------------------------------  
# Cleanup
# ------------------------------------------------  

data "external" "cleanup_unzipped_dir" {
  depends_on = [tfe_policy_set.workspace_scoped_policy_set]
  program = ["python3", "${path.module}/fetch_python.py"]

  query = {
    action     = "cleanup"
    unzip_dir  = local.unzipped_policy_dir
    run_id     = timestamp()
  }
}