# Dify Sandbox Python SDK

Python SDK for interacting with the Dify Sandbox API.

## Installation

```bash
pip install -e sdk/
```

Or install dependencies directly:

```bash
pip install requests
```

## Quick Start

```python
from dify_sandbox import DifySandboxClient

# Initialize client
client = DifySandboxClient(
    base_url="http://localhost:8194",
    api_key="your-api-key"
)

# Run Python code
result = client.run_python("print('Hello from sandbox!')")
print(result.stdout)  # Output: Hello from sandbox!

# Run Node.js code
result = client.run_nodejs("console.log('Hello from Node.js!')")
print(result.stdout)  # Output: Hello from Node.js!
```

## Features

- **Code Execution**: Run Python and Node.js code in a secure sandbox
- **File Operations**: Upload and download files to/from the sandbox
- **Dependency Management**: View and manage sandbox dependencies
- **Health Check**: Monitor sandbox server status

## API Reference

### Client Initialization

```python
client = DifySandboxClient(
    base_url="http://localhost:8194",  # Sandbox server URL
    api_key="your-api-key",             # API key for authentication
    timeout=30                          # Request timeout in seconds
)
```

### Code Execution

#### Run Python Code

```python
result = client.run_python(
    code="print('Hello, World!')",
    preload="",              # Optional: code to run before main code
    enable_network=False     # Optional: enable network access
)

print(result.stdout)    # Standard output
print(result.stderr)    # Standard error
print(result.exit_code) # Exit code (0 = success)
```

#### Run Node.js Code

```python
result = client.run_nodejs(
    code="console.log('Hello, World!')",
    preload="",
    enable_network=False
)

print(result.stdout)
print(result.stderr)
print(result.exit_code)
```

### File Operations

#### Upload File

```python
# Upload from file path
result = client.upload_file("/path/to/file.txt")
print(result.filename)  # Uploaded filename in sandbox
print(result.size)      # File size in bytes

# Upload from file object
with open("local_file.txt", "rb") as f:
    result = client.upload_file(f, filename="custom_name.txt")
```

#### Download File

```python
# Download to memory
content = client.download_file("sandbox_file.txt")

# Download to local file
client.download_file("sandbox_file.txt", save_path="local_copy.txt")
```

### Dependency Management

#### Get Dependencies

```python
deps = client.get_dependencies(language="python3")
print(deps.dependencies)  # List of installed packages
```

#### Update Dependencies

```python
response = client.update_dependencies(language="python3")
print(response.message)
```

#### Refresh Dependencies

```python
response = client.refresh_dependencies(language="python3")
print(response.message)
```

### Health Check

```python
if client.health_check():
    print("Sandbox is healthy")
else:
    print("Sandbox is not responding")
```

## Data Models

### RunCodeResponse

```python
@dataclass
class RunCodeResponse:
    stdout: str      # Standard output from code execution
    stderr: str      # Standard error from code execution
    exit_code: int   # Exit code (0 = success)
```

### UploadFileResponse

```python
@dataclass
class UploadFileResponse:
    filename: str  # Filename in sandbox
    size: int      # File size in bytes
```

### DependencyInfo

```python
@dataclass
class DependencyInfo:
    language: str       # Language (python3/nodejs)
    dependencies: list  # List of dependencies
```

### DifySandboxResponse

```python
@dataclass
class DifySandboxResponse:
    code: int       # Response code (0 = success)
    message: str    # Response message
    data: Any       # Response data
```

## Examples

See the `examples/` directory for complete usage examples:

- `basic_usage.py` - Basic code execution examples
- `file_operations.py` - File upload and download examples
- `dependency_management.py` - Dependency management examples

## Error Handling

The SDK raises exceptions for API errors:

```python
try:
    result = client.run_python("invalid code")
except Exception as e:
    print(f"Error: {e}")
```

## License

MIT License
