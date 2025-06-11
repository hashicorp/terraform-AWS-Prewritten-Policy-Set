module "policy_set_test" {
  source = "../pre-written-policy"

  name                         = "tushar_test_module_1"
  policy_github_repository     = "policy-library-CIS-Policy-Set-for-AWS-Terraform"
  policy_set_workspace_names   = ["vcs-testing"]
  tfe_organization             = "team-rnd-india-test-org"
}