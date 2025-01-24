const { privateKeyToAccount } = require('viem/accounts');
const config = require('./config.json');
const serializeBigInt = (key, value) => {
    if (key === "chainId") return Number(value);
    return typeof value === 'bigint' ? value.toString() : value;
};

const typedDataTemplate = {
    domain: {
        name: "Cartesi",
        version: "0.1.0",
        chainId: 0n, // To be set dynamically
        verifyingContract: "0x0000000000000000000000000000000000000000",
    },
    types: {
        EIP712Domain: [
            { name: "name", type: "string" },
            { name: "version", type: "string" },
            { name: "chainId", type: "uint256" },
            { name: "verifyingContract", type: "address" },
        ],
        CartesiMessage: [
            { name: "app", type: "address" },
            { name: "nonce", type: "uint64" },
            { name: "max_gas_price", type: "uint128" },
            { name: "data", type: "bytes" },
        ],
    },
    primaryType: "CartesiMessage",
    message: {},
};

const fetchNonceL2 = async (user, application, chainConfig) => {
    const response = await fetch(`${chainConfig.l2EIP712Url}/nonce`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ msg_sender: user, app_contract: application }),
    });
    if (!response.ok) {
        throw new Error(`Failed to fetch nonce: ${await response.text()}`);
    }
    const responseData = await response.json();
    return responseData.nonce;
};

const submitTransactionL2 = async (body, chainConfig) => {
    const response = await fetch(`${chainConfig.l2EIP712Url}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body, serializeBigInt),
    });
    if (!response.ok) {
        throw new Error(`submit to L2 failed: ${await response.text()}`);
    }
    return response.json();
};

const addTransactionL2 = async (chainId, appAddress, payload) => {
    const account1 = privateKeyToAccount(process.env.SENDER_PRIVATE_KEY)
    const account = account1.publicKey
    const chainConfig = config.chains[chainId];
    const nonce = await fetchNonceL2(account, appAddress, chainConfig);

    const typedData = { ...typedDataTemplate };
    typedData.domain.chainId = BigInt(parseInt(chainId.substring(2), 16));
    typedData.message = {
        app: appAddress,
        nonce: nonce,
        data: payload,
        max_gas_price: BigInt(10),
    };
    
    const signature = await account1.signTypedData({ account, ...typedData });
    const l2Data = {
        typedData,
        account,
        signature,
    };

    const response = await submitTransactionL2(l2Data, chainConfig);
    console.log(`Transaction submitted to L2. Transaction ID: ${response.id}`);
    console.log(typedData)
    return response.id;
};

(async () => {
    const chainId = process.env.CHAIN_ID || "0xaa36a7"; // sepolia
    const appAddress = process.env.APP_ADDRESS || "0x5a205fcb6947e200615b75c409ac0aa486d77649";
    const inputData = process.env.INPUT || "0xdeadbeef";

    try {
        await addTransactionL2(chainId, appAddress, inputData);
    } catch (error) {
        console.error("Error:", error);
    }
})();
