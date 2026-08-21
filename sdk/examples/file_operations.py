"""
File operations examples for Dify Sandbox SDK
"""

import os
import tempfile
from dify_sandbox import DifySandboxClient


def main():
    # Initialize client
    client = DifySandboxClient(
        base_url="http://localhost:8194",
        api_key="dify-sandbox"
    )
    
    # Create a temporary file for testing
    with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
        f.write("Hello from uploaded file!\n")
        f.write("This is a test file for Dify Sandbox.\n")
        temp_file = f.name
    
    try:
        # Upload file from path
        print("=== Upload File from Path ===")
        upload_result = client.upload_file(temp_file)
        print(f"Uploaded filename: {upload_result.filename}")
        print(f"File size: {upload_result.size} bytes")
        
        # Upload file from file object
        print("\n=== Upload File from File Object ===")
        with open(temp_file, 'rb') as f:
            upload_result2 = client.upload_file(f, filename="custom_name.txt")
        print(f"Uploaded filename: {upload_result2.filename}")
        print(f"File size: {upload_result2.size} bytes")
        
        # Download file to memory
        print("\n=== Download File to Memory ===")
        content = client.download_file(upload_result.filename)
        print(f"Downloaded content:\n{content.decode('utf-8')}")
        
        # Download file to local file
        print("\n=== Download File to Local File ===")
        download_path = "downloaded_file.txt"
        client.download_file(upload_result.filename, save_path=download_path)
        print(f"File downloaded to: {download_path}")
        
        # Verify downloaded file
        with open(download_path, 'r') as f:
            print(f"Content:\n{f.read()}")
        
        # Clean up
        if os.path.exists(download_path):
            os.remove(download_path)
            print(f"\nCleaned up: {download_path}")
    
    finally:
        # Clean up temp file
        if os.path.exists(temp_file):
            os.remove(temp_file)
            print(f"Cleaned up: {temp_file}")


if __name__ == "__main__":
    main()
