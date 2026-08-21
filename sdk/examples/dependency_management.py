"""
Dependency management examples for Dify Sandbox SDK
"""

from dify_sandbox import DifySandboxClient


def main():
    # Initialize client
    client = DifySandboxClient(
        base_url="http://localhost:8194",
        api_key="dify-sandbox"
    )
    
    # Get Python dependencies
    print("=== Python Dependencies ===")
    try:
        deps = client.get_dependencies(language="python3")
        print(f"Language: {deps.language}")
        print(f"Dependencies: {deps.dependencies}")
    except Exception as e:
        print(f"Error: {e}")
    
    # Update Python dependencies
    print("\n=== Update Python Dependencies ===")
    try:
        response = client.update_dependencies(language="python3")
        print(f"Status: {response.message}")
    except Exception as e:
        print(f"Error: {e}")
    
    # Refresh Python dependencies
    print("\n=== Refresh Python Dependencies ===")
    try:
        response = client.refresh_dependencies(language="python3")
        print(f"Status: {response.message}")
    except Exception as e:
        print(f"Error: {e}")


if __name__ == "__main__":
    main()
