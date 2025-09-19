# Warden Mainnet

This repository contains the resources and coordination process for launching the **Warden mainnet**.

---

## Launch Process

To join the Warden mainnet as a validator, please follow the steps below:

1. **Submit a Pull Request**

   - Create a PR to this repository.
   - Include your generated `gentx` JSON file in the `mainnet/gentx` folder.
   - Use the default filename produced by the `gentx` command (do not rename the file to prevent naming collisions).

2. **Genesis Preparation**

   - After all validator `gentx` files are collected, the Warden Labs team will build the final `genesis.json` file.

3. **Genesis Upload**

   - The finalized `genesis.json` will be published to the `mainnet/genesis` folder in this repository.

4. **Network Bootstrapping**

   - The Warden Labs team will start the initial node and share a list of `seed` and `persistent_peers` values for validators to use in their configurations.

5. **Validator Onboarding**
   - Once the Warden Labs team gives the go-ahead, all validators can bring their nodes online in a synchronized manner.

---

## Documentation

For details on how to become a validator and best practices for running nodes, refer to the Warden documentation:  
https://docs.wardenprotocol.org/operate-a-node/introduction

---

## Network Configuration

| Name      | Value | Description                             |
| --------- | ----- | --------------------------------------- |
| Chain ID  | `TBD` | Unique identifier of the Warden mainnet |
| Seed Node | `TBD` | Seed node to help discover other peers  |
