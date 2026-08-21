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

# Upload a Python script and run it from the sandbox upload directory
with open("hello.py", "rb") as f:
    uploaded = client.upload_file(f, filename="hello.py")

result = client.run_command(
    command="python3",
    args=[uploaded.filename],
    timeout=30,
)
print(result.stdout)
```

## Features

- **Code Execution**: Run Python and Node.js code in a secure sandbox
- **Command Execution**: Launch sandbox-approved binaries (e.g. `python3`,
  `node`) against previously uploaded files via the deny-list enforced
  `POST /v1/sandbox/run/command` endpoint
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

### Command Execution

Run a sandbox-approved binary against a file that was previously uploaded
to the sandbox via `upload_file`. The endpoint enforces a deny-list of
dangerous commands (shells, `rm`, `sudo`, package managers, …) and rejects
any argument containing shell metacharacters so the request can never
accidentally spawn a shell.

```python
# 1. Upload the script you want to execute
with open("hello.py", "rb") as f:
    uploaded = client.upload_file(f, filename="hello.py")

# 2. Invoke python3 with the uploaded file as an argument
result = client.run_command(
    command="python3",            # Command basename (resolved via PATH)
    args=[uploaded.filename],     # Arguments; cannot contain shell metachars
    work_dir="",                  # "" or "." → sandbox upload_dir; or a relative subdir
    timeout=30,                   # 0 → use the sandbox worker timeout
    enable_network=False,         # Must respect global enable_network setting
)

print(result.stdout)
print(result.stderr)
print(result.exit_code)
```

#### `work_dir` rules

| Input                | Result                                                |
|----------------------|-------------------------------------------------------|
| `""` or `"."`        | Resolved to the sandbox `upload_dir` itself           |
| `"scripts"`          | Resolved to `<upload_dir>/scripts`                    |
| `<upload_dir>` (absolute) | Resolved to `upload_dir` itself                  |
| `/etc`, `../etc`, …  | Rejected with `work_dir is invalid: ...`              |

#### Deny-list

The deny-list is the union of a built-in default (shells, `rm`, `sudo`,
`apt`, `pip3`, `npm`, …) and the operator-configured
`blocked_commands` list. User configuration can only add entries — never
remove them. Commands that match the deny-list, or arguments containing
shell metacharacters (`|`, `&`, `;`, `<`, `>`, `` ` ``, `$`, `*`, `?`,
`{`, `}`, `~`, `!`, `#`, quotes, …) are rejected with a 400 before any
process is spawned.

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
    error: str       # Sandbox-side error message (empty on success)
```

### RunCommandResponse

```python
@dataclass
class RunCommandResponse:
    stdout: str      # Standard output from the executed command
    stderr: str      # Standard error from the executed command
    exit_code: int   # Exit code (0 = success)
    error: str       # Sandbox-side error message (empty on success)
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

## End-to-end Example: Upload a Script, Then Run It

```python
from dify_sandbox import DifySandboxClient

client = DifySandboxClient(base_url="http://localhost:8194", api_key="dify-sandbox")

# Local script we want to run inside the sandbox
script = b"""
import sys
print("hello from the sandbox!")
print("args:", sys.argv[1:])
"""

# 1. Upload the script — the server stores it in upload_dir and returns the
#    filename it used (a UUID is appended to avoid collisions).
with open("hello.py", "wb") as f:
    f.write(script)

uploaded = client.upload_file("hello.py")
print("uploaded as:", uploaded.filename)

# 2. Run python3 with the uploaded file as its argument.
result = client.run_command(
    command="python3",
    args=[uploaded.filename],
    timeout=10,
)

assert result.exit_code == 0, result.stderr
print(result.stdout)
```

## Examples

See the `examples/` directory for complete usage examples:

- `basic_usage.py` - Basic code execution examples
- `file_operations.py` - File upload and download examples
- `dependency_management.py` - Dependency management examples
- `command_execution.py` - Upload a script and run it through the deny-list-enforced `run_command` endpoint

## Error Handling

The SDK raises exceptions for API errors:

```python
try:
    result = client.run_python("invalid code")
except Exception as e:
    print(f"Error: {e}")
```

`run_command` raises the same kind of exception when the deny-list rejects
the command, when an argument contains shell metacharacters, or when the
work directory is invalid — the exception message is the human-readable
reason returned by the sandbox.

## License

MIT License
