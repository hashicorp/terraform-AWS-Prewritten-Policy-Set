data "external" "fetch" {
  program = ["python3", "${path.module}/testing.py"]

  query = {
    action = "download"
    owner = "hashicorp"
    repo  = "policy-library-CIS-Policy-Set-for-AWS-Terraform"
    name  = "testing_python"
  }
}
