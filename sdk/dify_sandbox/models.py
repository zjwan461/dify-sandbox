"""
Dify Sandbox SDK - Data Models

This module contains data models for the Dify Sandbox SDK.
"""

from dataclasses import dataclass
from typing import Optional, Any, Dict


@dataclass
class DifySandboxResponse:
    """Base response from Dify Sandbox API"""
    code: int
    message: str
    data: Optional[Any] = None

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "DifySandboxResponse":
        return cls(
            code=data.get("code", -1),
            message=data.get("message", ""),
            data=data.get("data")
        )

    @property
    def is_success(self) -> bool:
        return self.code == 0


@dataclass
class RunCodeResponse:
    """Response from code execution endpoint"""
    stdout: str
    stderr: str
    error: str
    exit_code: int

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RunCodeResponse":
        return cls(
            stdout=data.get("stdout", ""),
            stderr=data.get("stderr", ""),
            error=data.get("error", ""),
            exit_code=data.get("exit_code", -1)
        )

    @property
    def is_success(self) -> bool:
        return self.exit_code == 0 and not self.error


@dataclass
class UploadFileResponse:
    """Response from file upload endpoint"""
    filename: str
    size: int

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "UploadFileResponse":
        return cls(
            filename=data.get("filename", ""),
            size=data.get("size", 0)
        )


@dataclass
class DependencyInfo:
    """Information about installed dependencies"""
    language: str
    dependencies: list

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "DependencyInfo":
        return cls(
            language=data.get("language", ""),
            dependencies=data.get("dependencies", [])
        )
