# Warden Protocol Networks - GenTx Validation with Dagger

This repository uses [Dagger](https://dagger.io) to provide consistent GenTx validation both in CI and locally. Dagger ensures that the same validation environment runs everywhere - your laptop, CI/CD, and any other environment.

## Prerequisites

### Install Dagger CLI

#### macOS
```bash
brew install dagger/tap/dagger
```

#### Linux/WSL
```bash
curl -fsSL https://dl.dagger.io/dagger/install.sh | BIN_DIR=$HOME/.local/bin sh
```

#### Windows (PowerShell)
```powershell
Invoke-WebRequest -UseBasicParsing -Uri https://dl.dagger.io/dagger/install.ps1 | Invoke-Expression
```

### Verify Installation
```bash
dagger version
```

## Local Development Commands

### 1. Validate GenTx Files

Validate all mainnet GenTx files:
```bash
dagger call validate-gentx --source .
```

Validate with specific parameters:
```bash
dagger call validate-gentx \
  --source . \
  --network mainnet \
  --wardend-version v0.7.0-rc3 \
  --go-version 1.24
```

### 2. Run Validation with Detailed Output

Get detailed validation results formatted for local development:
```bash
dagger call run-local-validation --source .
```

With custom parameters:
```bash
dagger call run-local-validation \
  --source . \
  --network mainnet \
  --wardend-version v0.7.0-rc3 \
  --go-version 1.24
```

### 3. Test the check-genesis Tool

Test that the check-genesis tool builds and runs correctly:
```bash
dagger call test-check-genesis-tool --source .
```

With custom Go version:
```bash
dagger call test-check-genesis-tool \
  --source . \
  --go-version 1.24
```

## How It Works

### Architecture

1. **Dagger Module**: Located in `ci/main.go`, defines all validation functions
2. **Configuration**: `dagger.json` configures the Dagger module
3. **GitHub Actions**: Uses Dagger CLI to run the same validation logic as locally

### Validation Process

1. **Build Phase**: 
   - Creates a Go container
   - Builds the `check-genesis` tool from `utils/check-genesis/`
   - Verifies the tool is executable

2. **Validation Phase**:
   - Creates a wardend container with the official Docker image
   - Mounts the built `check-genesis` tool
   - For each GenTx file:
     - Sets up isolated validation environment
     - Copies genesis file and GenTx file
     - Runs validation inside the wardend container

3. **Results**:
   - Returns structured results with pass/fail status
   - Provides detailed error messages for failures

### Benefits of Dagger

✅ **Consistency**: Same environment locally and in CI  
✅ **Reproducibility**: Containerized execution eliminates "works on my machine"  
✅ **Speed**: Efficient caching and parallel execution  
✅ **Debugging**: Easy to run locally when CI fails  
✅ **Isolation**: Each validation runs in a clean container  

## Debugging Failed Validations

### 1. Run Local Validation
```bash
dagger call run-local-validation --source .
```

### 2. Check Specific Tool Build
```bash
dagger call test-check-genesis-tool --source .
```

### 3. Debug with Shell Access
To get shell access to the validation environment:
```bash
dagger call validate-gentx --source . --terminal
```

## Development Workflow

### Adding New Validation Logic

1. Edit `ci/main.go` to add new validation functions
2. Test locally: `dagger call your-new-function --source .`  
3. Update GitHub Actions workflow if needed
4. Commit changes - CI will use the updated logic automatically

### Testing Changes

Always test your changes locally before pushing:
```bash
# Test the build process
dagger call test-check-genesis-tool --source .

# Test validation
dagger call run-local-validation --source .

# Test with different parameters
dagger call validate-gentx --source . --wardend-version v0.6.0
```

## Troubleshooting

### Common Issues

**1. "dagger: command not found"**
- Install Dagger CLI using the instructions above
- Ensure it's in your PATH

**2. "failed to get gentx files"** 
- Ensure you're running from the repository root
- Check that `mainnet/gentx/` directory exists and contains `.json` files

**3. "validation failed"**
- Run with verbose output: `dagger call run-local-validation --source .`
- Check individual file issues in the detailed output

**4. "wardend image pull failed"**
- Check your internet connection
- Verify the wardend version exists: `docker pull ghcr.io/warden-protocol/wardenprotocol/wardend:v0.7.0-rc3`

### Getting Help

- **Dagger Documentation**: https://docs.dagger.io
- **Dagger Community**: https://discord.gg/ufnyBtc8uY
- **Repository Issues**: Create an issue in this repository for validation-specific problems

## Advanced Usage

### Custom Wardend Version

Test against a different wardend version:
```bash
dagger call validate-gentx \
  --source . \
  --wardend-version v0.6.0
```

### Different Go Version

Use a different Go version for building tools:
```bash
dagger call validate-gentx \
  --source . \
  --go-version 1.21
```

### Integration with Other Tools

The Dagger functions can be integrated into other workflows:

```bash
# Use in scripts
if dagger call validate-gentx --source .; then
  echo "Validation passed!"
  # Deploy or continue with other tasks
else
  echo "Validation failed!"
  exit 1
fi
```

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/validate-gentx.yml`) uses the same Dagger functions:

1. Installs Dagger CLI
2. Runs `dagger call validate-gentx --source .`
3. Provides PR comments with results
4. Includes instructions for local reproduction

This ensures perfect parity between local development and CI environments.
