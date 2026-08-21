"""
Dify Sandbox SDK - Main Client

This module provides the main client for interacting with the Dify Sandbox API.
"""

import os
from typing import Optional, Union, BinaryIO
import requests

from .models import (
    DifySandboxResponse,
    RunCodeResponse,
    UploadFileResponse,
    DependencyInfo,
)


class DifySandboxClient:
    """
    Client for interacting with Dify Sandbox API.
    
    Example:
        client = DifySandboxClient(
            base_url="http://localhost:8194",
            api_key="dify-sandbox"
        )
        
        # Run Python code
        result = client.run_python("print('Hello, World!')")
        print(result.stdout)
        
        # Upload a file
        with open("test.txt", "rb") as f:
            upload_result = client.upload_file(f, "test.txt")
        
        # Download a file
        content = client.download_file(upload_result.data.filename)
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8194",
        api_key: str = "dify-sandbox",
        timeout: int = 30,
    ):
        """
        Initialize the Dify Sandbox client.
        
        Args:
            base_url: Base URL of the Dify Sandbox server
            api_key: API key for authentication (X-Api-Key header)
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({"X-Api-Key": self.api_key})

    def _request(
        self,
        method: str,
        path: str,
        json: Optional[dict] = None,
        files: Optional[dict] = None,
        stream: bool = False,
    ) -> requests.Response:
        """
        Make an HTTP request to the API.
        
        Args:
            method: HTTP method (GET, POST, etc.)
            path: API path (will be appended to base_url)
            json: JSON data to send
            files: Files to upload
            stream: Whether to stream the response
            
        Returns:
            requests.Response object
            
        Raises:
            requests.HTTPError: If the request fails
        """
        url = f"{self.base_url}{path}"
        response = self.session.request(
            method=method,
            url=url,
            json=json,
            files=files,
            timeout=self.timeout,
            stream=stream,
        )
        response.raise_for_status()
        return response

    def health_check(self) -> bool:
        """
        Check if the sandbox server is healthy.
        
        Returns:
            True if server is healthy, False otherwise
        """
        try:
            response = self._request("GET", "/health")
            return response.status_code == 200
        except requests.RequestException:
            return False

    def run_python(
        self,
        code: str,
        preload: str = "",
        enable_network: bool = False,
    ) -> RunCodeResponse:
        """
        Run Python code in the sandbox.
        
        Args:
            code: Python code to execute
            preload: Optional preload code to run before main code
            enable_network: Whether to enable network access
            
        Returns:
            RunCodeResponse with execution results
        """
        return self._run_code("python3", code, preload, enable_network)

    def run_nodejs(
        self,
        code: str,
        preload: str = "",
        enable_network: bool = False,
    ) -> RunCodeResponse:
        """
        Run Node.js code in the sandbox.
        
        Args:
            code: Node.js code to execute
            preload: Optional preload code to run before main code
            enable_network: Whether to enable network access
            
        Returns:
            RunCodeResponse with execution results
        """
        return self._run_code("nodejs", code, preload, enable_network)

    def _run_code(
        self,
        language: str,
        code: str,
        preload: str,
        enable_network: bool,
    ) -> RunCodeResponse:
        """
        Internal method to run code in the sandbox.
        
        Args:
            language: Programming language ("python3" or "nodejs")
            code: Code to execute
            preload: Preload code
            enable_network: Enable network access
            
        Returns:
            RunCodeResponse with execution results
        """
        payload = {
            "language": language,
            "code": code,
            "preload": preload,
            "enable_network": enable_network,
        }
        response = self._request("POST", "/v1/sandbox/run", json=payload)
        result = DifySandboxResponse.from_dict(response.json())
        
        if not result.is_success:
            raise Exception(f"API error: {result.message}")
        
        return RunCodeResponse.from_dict(result.data)

    def get_dependencies(self, language: str = "python3") -> DependencyInfo:
        """
        Get list of installed dependencies.
        
        Args:
            language: Programming language ("python3" or "nodejs")
            
        Returns:
            DependencyInfo with list of dependencies
        """
        response = self._request(
            "GET", "/v1/sandbox/dependencies", json={"language": language}
        )
        result = DifySandboxResponse.from_dict(response.json())
        
        if not result.is_success:
            raise Exception(f"API error: {result.message}")
        
        return DependencyInfo.from_dict(result.data)

    def update_dependencies(self, language: str = "python3") -> DifySandboxResponse:
        """
        Update dependencies for the specified language.
        
        Args:
            language: Programming language ("python3" or "nodejs")
            
        Returns:
            DifySandboxResponse with update result
        """
        response = self._request(
            "POST", "/v1/sandbox/dependencies/update", json={"language": language}
        )
        result = DifySandboxResponse.from_dict(response.json())
        
        if not result.is_success:
            raise Exception(f"API error: {result.message}")
        
        return result

    def refresh_dependencies(self, language: str = "python3") -> DifySandboxResponse:
        """
        Refresh dependencies for the specified language.
        
        Args:
            language: Programming language ("python3" or "nodejs")
            
        Returns:
            DifySandboxResponse with refresh result
        """
        response = self._request(
            "GET", "/v1/sandbox/dependencies/refresh", json={"language": language}
        )
        result = DifySandboxResponse.from_dict(response.json())
        
        if not result.is_success:
            raise Exception(f"API error: {result.message}")
        
        return result

    def upload_file(
        self,
        file: Union[BinaryIO, str],
        filename: Optional[str] = None,
    ) -> UploadFileResponse:
        """
        Upload a file to the sandbox.
        
        Args:
            file: File object or file path to upload
            filename: Optional filename (required if file is a file object)
            
        Returns:
            UploadFileResponse with uploaded file info
            
        Example:
            # Upload from file path
            result = client.upload_file("/path/to/file.txt")
            
            # Upload from file object
            with open("file.txt", "rb") as f:
                result = client.upload_file(f, "file.txt")
        """
        if isinstance(file, str):
            # file is a path
            if not os.path.exists(file):
                raise FileNotFoundError(f"File not found: {file}")
            
            if filename is None:
                filename = os.path.basename(file)
            
            with open(file, "rb") as f:
                return self._upload_file_object(f, filename)
        else:
            # file is a file object
            if filename is None:
                raise ValueError("filename is required when file is a file object")
            return self._upload_file_object(file, filename)

    def _upload_file_object(
        self,
        file: BinaryIO,
        filename: str,
    ) -> UploadFileResponse:
        """
        Internal method to upload a file object.
        
        Args:
            file: File object to upload
            filename: Name of the file
            
        Returns:
            UploadFileResponse with uploaded file info
        """
        files = {"file": (filename, file)}
        response = self._request("POST", "/v1/sandbox/file/upload", files=files)
        result = DifySandboxResponse.from_dict(response.json())
        
        if not result.is_success:
            raise Exception(f"API error: {result.message}")
        
        return UploadFileResponse.from_dict(result.data)

    def download_file(
        self,
        filename: str,
        save_path: Optional[str] = None,
    ) -> Union[bytes, str]:
        """
        Download a file from the sandbox.
        
        Args:
            filename: Name of the file to download
            save_path: Optional path to save the file. If not provided, returns bytes.
            
        Returns:
            File content as bytes if save_path is None, otherwise the save_path
            
        Example:
            # Download to bytes
            content = client.download_file("uploaded_file.txt")
            
            # Download to file
            client.download_file("uploaded_file.txt", "downloaded.txt")
        """
        payload = {"filename": filename}
        response = self._request(
            "POST", "/v1/sandbox/file/download", json=payload, stream=True
        )
        
        if save_path:
            with open(save_path, "wb") as f:
                for chunk in response.iter_content(chunk_size=8192):
                    f.write(chunk)
            return save_path
        else:
            return response.content
