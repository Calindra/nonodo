# Espresso + Cartesi

Diagram:

```mermaid
sequenceDiagram
    actor User
    participant Frontend as Front end
    participant CartesiNode as Cartesi Node
    participant Espresso
    participant InputBox

    User->>Frontend: Enter address and payload
    Frontend->>CartesiNode: Query nonce
    CartesiNode-->>Frontend: Return Nonce
    Frontend->>User: Ask user for EIP712 sign
    User->>Frontend: Sign
    Frontend->>CartesiNode: Submit message and signature
    CartesiNode->>Espresso: Submit to namespace
    CartesiNode-->>Frontend: Return input id
    Frontend-->>User: Display input id
    CartesiNode->>Espresso: Fetch header and extract L1 finalized nth-block
    User-->>InputBox: Sign and submit tx to L1
    CartesiNode-->>InputBox: If L1 finalized is updated fetch from inputbox
    CartesiNode->>Espresso: Fetch txs in the Espresso block filtered by namespace
    User->>CartesiNode: Query outputs by using input id
```

## Running Nonodo for local development

```bash
go build . && ./nonodo
```

## Running Nonodo for testnet development with Espresso

```bash
go build . && ./nonodo --contracts-application-address 0x70ac08179605AF2D9e75782b8DEcDD3c22aA4D0C --sequencer espresso -d --from-block <blocknumber> --rpc-url <sepolia> --espresso-url <espresso-url>
```
