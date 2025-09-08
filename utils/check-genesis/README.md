# GenTx Validation

This repository includes automated GenTx (Genesis Transaction) validation using the `check-genesis` tool in `utils/check-genesis/`.

## Automated Validation

### When Validation Runs

The GenTx validation workflow runs automatically in the following scenarios:

1. **Pull Requests**: When GenTx files are added or modified in PRs
2. **Main Branch**: When GenTx files are pushed to the main branch
3. **Manual Dispatch**: Can be triggered manually for specific networks or files

### What Gets Validated

For each GenTx file, the validation process:

1. ✅ **File Existence**: Checks that the GenTx file exists and is readable
2. ✅ **JSON Format**: Validates the GenTx file has proper JSON structure
3. ✅ **Fee Validation**: Ensures the fee meets minimum requirements (180000000000000000 award)
4. ✅ **Genesis Integration**: Tests integration with the network's genesis file
5. ✅ **Node Startup**: Verifies the genesis can be used to start a node without panics

### Validation Results

The workflow will:

- ✅ **Pass**: Comment on PRs with success status
- ❌ **Fail**: Comment on PRs with failure details and common troubleshooting tips
- 📋 **Logs**: Upload detailed validation logs as artifacts

## Manual Validation

### Using GitHub Actions

You can manually validate a specific GenTx file:

1. Go to **Actions** → **Manual GenTx Validation**
2. Click **Run workflow**
3. Fill in the parameters:
   - **GenTx file**: Path to the GenTx file (e.g., `testnets/alfama/gentx/gentx-validator-1.json`)
   - **Network**: Network name (e.g., `alfama`, `chiado`)
   - **Network type**: Choose `testnets` or `mainnet`

### Using the Tool Locally

You can also run the validation tool locally:

```bash
# Navigate to the repository root
cd /path/to/networks

# Build the check-genesis tool
cd utils/check-genesis
go build -o check-genesis ./check-genesis.go

# Prepare test environment
mkdir -p /tmp/gentx-test
cd /tmp/gentx-test

# Copy the network's genesis file as init_genesis.json
cp /path/to/networks/testnets/alfama/genesis.json ./init_genesis.json

# Run validation
/path/to/networks/utils/check-genesis/check-genesis /path/to/networks/testnets/alfama/gentx/gentx-validator-1.json
```

## Requirements

### Prerequisites

- **wardend**: The validation tool requires `wardend` binary (v0.6.5 or compatible)
- **Go**: Version 1.21 or later for building the check-genesis tool
- **Genesis File**: Each network must have a `genesis.json` file in its directory

### GenTx File Requirements

Your GenTx file must:

1. **Valid JSON**: Be a properly formatted JSON file
2. **Minimum Fee**: Include a fee of at least `180000000000000000 award`
3. **Proper Structure**: Follow the standard Cosmos SDK GenTx format
4. **Compatible**: Be compatible with the target network's genesis file

### Expected Directory Structure

```
testnets/
├── alfama/
│   ├── genesis.json          # ← Required
│   └── gentx/
│       ├── gentx-validator-1.json
│       └── gentx-validator-2.json
├── chiado/
│   ├── genesis.json          # ← Required  
│   └── gentx/
│       └── gentx-validator-1.json
└── ...

mainnet/
├── network-name/
│   ├── genesis.json          # ← Required
│   └── gentx/
│       └── gentx-validator-1.json
└── ...
```

## Troubleshooting

### Common Issues

#### ❌ Fee Too Low
```
Error: gentx fee is less than minimum required fee
```
**Solution**: Ensure your GenTx includes a fee of at least `180000000000000000 award`.

#### ❌ Invalid JSON
```
Error: failed to parse gentx JSON
```
**Solution**: Validate your JSON format using a JSON validator or linter.

#### ❌ Missing Genesis File
```
Error: Genesis file not found
```
**Solution**: Ensure the network has a `genesis.json` file in its directory.

#### ❌ Node Startup Failure
```
Error: node start test failed
```
**Solution**: Check the uploaded logs for detailed error messages. Common causes include:
- Invalid validator configuration
- Network parameter mismatches
- Insufficient system resources

### Getting Help

1. **Check Logs**: Download validation logs from the GitHub Actions artifacts
2. **Review GenTx**: Ensure your GenTx file follows the standard format
3. **Compare Working Examples**: Look at existing validated GenTx files in the repository
4. **Open Issue**: Create an issue if you believe there's a problem with the validation process

## Validation Tool Details

The `check-genesis` tool performs these steps:

1. **Setup**: Creates temporary directories and configuration
2. **Fee Check**: Validates the GenTx fee meets minimum requirements
3. **Genesis Collection**: Uses `wardend genesis collect-gentxs` to integrate the GenTx
4. **Genesis Validation**: Runs `wardend genesis validate-genesis`
5. **Node Test**: Starts a node temporarily to ensure no panics occur
6. **Cleanup**: Provides detailed logs and cleanup

### Environment Variables

The tool uses these default configurations:

- `WARDEND`: `wardend` (binary name)
- `NETWORK`: `barra_9191-1` (chain ID)
- `TIMEOUT`: `60` seconds (node test timeout)
- `REQ_FEE`: `180000000000000000` (minimum fee in award)

## Contributing

When contributing GenTx files:

1. **Test Locally**: Run the validation tool locally before submitting
2. **Follow Structure**: Place files in the correct network directory
3. **Check Fees**: Ensure your fee meets the minimum requirements
4. **Submit PR**: The automated validation will run on your PR

The validation process helps ensure network stability and prevents common issues during genesis setup.
