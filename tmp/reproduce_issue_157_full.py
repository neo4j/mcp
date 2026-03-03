#!/usr/bin/env python3
"""
This follows the exact steps from the issue, but compiles the binary locally
instead of downloading from GitHub releases.

Setup (from project root):
    # 1. Create and activate virtual environment:
    python3 -m venv tmp/venv
    source tmp/venv/bin/activate

    # 2. Install dependencies:
    pip install langchain-mcp-adapters langchain-openai langgraph langchain

    # 3. Build the binary:
    go build -o bin/neo4j-mcp ./cmd/neo4j-mcp

    # 4. Set your OpenAI API key:
    export OPENAI_API_KEY=""

    # 5. Run this script:
    python tmp/reproduce_issue_157_full.py
"""

import asyncio
import base64
import os
import subprocess
import sys
import time


# Configuration - using the demo database from the issue
NEO4J_URI = os.environ.get("NEO4J_URI", "bolt://localhost:7687")
NEO4J_DATABASE = os.environ.get("NEO4J_DATABASE", "neo4j")
NEO4J_USERNAME = os.environ.get("NEO4J_USERNAME", "neo4j")
NEO4J_PASSWORD = os.environ.get("NEO4J_PASSWORD", "password")
MCP_PORT = os.environ.get("MCP_PORT", "8000")


def build_binary():
    print("Building neo4j-mcp binary...")
    result = subprocess.run(
        ["go", "build", "-o", "bin/neo4j-mcp", "./cmd/neo4j-mcp"],
        capture_output=True,
        text=True
    )
    if result.returncode != 0:
        print(f"Build failed: {result.stderr}")
        sys.exit(1)
    print("Binary built successfully")


def start_server():
    print(f"\nStarting MCP server on port {MCP_PORT}...")
    
    env = os.environ.copy()
    env["NEO4J_URI"] = NEO4J_URI
    env["NEO4J_DATABASE"] = NEO4J_DATABASE
    env["NEO4J_MCP_TRANSPORT"] = "http"
    env["NEO4J_MCP_HTTP_PORT"] = MCP_PORT
    
    # Start server in background (matching the issue's approach)
    process = subprocess.Popen(
        ["./bin/neo4j-mcp"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=env
    )
    
    # Give server time to start
    time.sleep(2)
    
    # Check if server started successfully
    if process.poll() is not None:
        stdout, stderr = process.communicate()
        print(f"Server failed to start!")
        print(f"stdout: {stdout.decode()}")
        print(f"stderr: {stderr.decode()}")
        sys.exit(1)
    
    print(f"Server started (PID: {process.pid})")
    return process


async def run_langchain_agent():
     from langchain_openai import ChatOpenAI
    from langchain_mcp_adapters.client import MultiServerMCPClient
    from langchain.agents import create_agent
 
    
    model =  ChatOpenAI(model="gpt-5.1")
    # Credentials passed via bearer auth (matching the issue)
    credentials = base64.b64encode(
        f"{NEO4J_USERNAME}:{NEO4J_PASSWORD}".encode()
    ).decode()
    
    cypher_mcp_config = {
        "neo4j-database": {
            "transport": "streamable_http",
            "url": f"http://localhost:{MCP_PORT}/mcp",
            "headers": {
                "Authorization": f"Basic {credentials}"
            },
        }
    }
    client = MultiServerMCPClient(cypher_mcp_config)
    mcp_tools = await client.get_tools()

    system_prompt = """
    You are a helpful assistant with access to a Neo4j graph database containing company data.
    Use the available tools to query the database and answer questions.
    """

    agent = create_agent(model, mcp_tools, system_prompt=system_prompt)

    prompt = "What's the schema of the database?"

    async for event in agent.astream(
        {"messages": [{"role": "user", "content": prompt}]},
        stream_mode="values",
    ):
        event["messages"][-1].pretty_print()
    
    


def main():
    server_process = None
    
    try:
        build_binary()
    
        server_process = start_server()
        
        print("Running LangChain MCP adapter test")
        
        success = asyncio.run(run_langchain_agent())
        
        return 0 if success else 1
        
    except KeyboardInterrupt:
        print("\n\nInterrupted by user")
        return 130
    finally:
        # Cleanup: stop the server
        if server_process:
            print("\nStopping MCP server...")
            server_process.terminate()
            server_process.wait(timeout=5)
            print("Server stopped")


if __name__ == "__main__":
    sys.exit(main())
