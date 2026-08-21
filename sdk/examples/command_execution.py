"""
Command execution examples for Dify Sandbox SDK.

This example demonstrates the typical workflow described in the README:
upload a script to the sandbox, then run it with the deny-list-enforced
``run_command`` endpoint. The endpoint rejects shells, ``rm``, ``sudo``,
package managers and any argument containing shell metacharacters, so
the upload-then-run pattern is the only supported way to invoke user
code through this surface.
"""

import os
import tempfile

from dify_sandbox import DifySandboxClient


def main():
    client = DifySandboxClient(
        base_url="http://localhost:8194",
        api_key="dify-sandbox",
    )

    if not client.health_check():
        print("Sandbox is not healthy; aborting")
        return

    # ---- 1. Upload a Python script --------------------------------------
    script_body = (
        b"import sys\n"
        b"print('hello from the sandbox!')\n"
        b"print('argv:', sys.argv[1:])\n"
    )
    with tempfile.NamedTemporaryFile(suffix=".py", delete=False) as tmp:
        tmp.write(script_body)
        local_path = tmp.name
    try:
        uploaded = client.upload_file(local_path)
        print(f"Uploaded script as: {uploaded.filename} "
              f"({uploaded.size} bytes)")

        # ---- 2. Run python3 against the uploaded file --------------------
        result = client.run_command(
            command="python3",
            args=[uploaded.filename],
            timeout=10,
        )
        print(f"exit_code: {result.exit_code}")
        print(f"stdout:\n{result.stdout}")
        if result.stderr:
            print(f"stderr:\n{result.stderr}")
        if result.error:
            print(f"sandbox error: {result.error}")

        # ---- 3. Demonstrate that blocked commands are rejected -----------
        for blocked in ("rm", "sh", "bash"):
            try:
                client.run_command(command=blocked, args=["-rf", "/"])
            except Exception as e:
                print(f"blocked {blocked!r} as expected: {e}")

        # ---- 4. Demonstrate that shell metacharacters are rejected -------
        try:
            client.run_command(
                command="python3",
                args=[f"{uploaded.filename}; rm -rf /"],
            )
        except Exception as e:
            print(f"metachar rejected as expected: {e}")
    finally:
        os.unlink(local_path)


if __name__ == "__main__":
    main()