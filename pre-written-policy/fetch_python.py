import os
import sys
import json
import shutil
import requests
import zipfile
from io import BytesIO

def download_file(file_path, url):
    response = requests.get(url, stream=True)
    response.raise_for_status()
    with open(file_path, 'wb') as f:
        for chunk in response.iter_content(chunk_size=8192):
            f.write(chunk)

def unzip_file(zip_path, dest_dir):
    with zipfile.ZipFile(zip_path, 'r') as zip_ref:
        zip_ref.extractall(dest_dir)

def move_nested_contents(temp_dir, dest_dir):
    entries = os.listdir(temp_dir)
    if not entries:
        raise Exception("No folders found in temp dir")

    first_level = os.path.join(temp_dir, entries[0])
    if not os.path.isdir(first_level):
        raise Exception("Expected a directory at first level")

    for item in os.listdir(first_level):
        src_path = os.path.join(first_level, item)
        dst_path = os.path.join(dest_dir, item)
        shutil.move(src_path, dst_path)

def handle_download(input_data):
    var_name = input_data["name"]
    policy_owner = input_data["owner"]
    policy_repo = input_data["repo"]

    download_dir = f"downloads-{var_name}"
    unzip_dir = f"unzipped-{var_name}"
    temp_dir = f"temp_unzip-{var_name}"

    os.makedirs(download_dir, exist_ok=True)
    os.makedirs(unzip_dir, exist_ok=True)
    os.makedirs(temp_dir, exist_ok=True)

    tags_url = f"https://api.github.com/repos/{policy_owner}/{policy_repo}/tags"
    res = requests.get(tags_url)
    if res.status_code != 200:
        raise Exception(f"GitHub API error: {res.status_code}")

    tags = res.json()
    if not tags:
        raise Exception("No tags found.")

    latest_zip_url = tags[0]["zipball_url"]
    latest_tag = tags[0]["name"]
    latest_zip_path = os.path.join(download_dir, f"{latest_tag}.zip")

    # Download the zip file
    download_file(latest_zip_path, latest_zip_url)

    # Unzip it
    unzip_file(latest_zip_path, temp_dir)

    # Move nested contents
    move_nested_contents(temp_dir, unzip_dir)

    # Cleanup
    shutil.rmtree(temp_dir)
    shutil.rmtree(download_dir)

    output = {
        "message": "Download and extraction completed successfully",
        "latest_tag": latest_tag,
        "unzip_dir": unzip_dir
    }
    print(json.dumps(output))

def handle_cleanup(input_data):
    unzip_dir = input_data["unzip_dir"]
    try:
        shutil.rmtree(unzip_dir)
    except Exception as e:
        print(json.dumps({"error": f"failed to cleanup: {str(e)}"}), file=sys.stderr)
        sys.exit(1)

    print(json.dumps({"message": "cleanup successful"}))

def main():
    try:
        input_data = json.load(sys.stdin)
        action = input_data.get("action")

        if action == "download":
            handle_download(input_data)
        elif action == "cleanup":
            handle_cleanup(input_data)
        else:
            print(json.dumps({"error": f"unknown action: {action}"}), file=sys.stderr)
            sys.exit(1)

    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
