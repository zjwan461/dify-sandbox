"""
Basic usage examples for Dify Sandbox SDK
"""

from dify_sandbox import DifySandboxClient


def main():
    # Initialize client
    client = DifySandboxClient(
        base_url="http://localhost:8194",
        api_key="dify-sandbox"
    )
    
    # Check health
    print("Health check:", client.health_check())
    
    # Run Python code
    print("\n=== Running Python Code ===")
    python_code = """
import sys
print("Hello from Python sandbox!")
print(f"Python version: {sys.version}")
result = 2 + 2
print(f"2 + 2 = {result}")
"""
    result = client.run_python(python_code)
    print(f"Exit code: {result.exit_code}")
    print(f"Stdout: {result.stdout}")
    if result.stderr:
        print(f"Stderr: {result.stderr}")
    
    # Run Node.js code
    print("\n=== Running Node.js Code ===")
    nodejs_code = """
const os = require('os');
console.log('Hello from Node.js sandbox!');
console.log(`Node version: ${process.version}`);
console.log(`Platform: ${os.platform()}`);
const result = 3 * 4;
console.log(`3 * 4 = ${result}`);
"""
    result = client.run_nodejs(nodejs_code)
    print(f"Exit code: {result.exit_code}")
    print(f"Stdout: {result.stdout}")
    if result.stderr:
        print(f"Stderr: {result.stderr}")
    
    # Run code with network enabled
    print("\n=== Running Code with Network ===")
    python_network_code = """
import urllib.request
try:
    response = urllib.request.urlopen('https://httpbin.org/get', timeout=5)
    print(f"HTTP Status: {response.status}")
except Exception as e:
    print(f"Network error: {e}")
"""
    result = client.run_python(python_network_code, enable_network=True)
    print(f"Exit code: {result.exit_code}")
    print(f"Stdout: {result.stdout}")


if __name__ == "__main__":
    main()
