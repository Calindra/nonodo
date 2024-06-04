import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '2s', target: 2 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        checks: ['rate>0.9'],
    }
};

const GRAPHQL_ENDPOINT = 'http://localhost:8080/graphql'

function testVoucherNotFound() {
    const payload = JSON.stringify({
        query: "query { voucher(voucherIndex: 0, inputIndex: 0) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testVoucherNotFound is status 200': (r) => r.status === 200,
        'testVoucherNotFound response body contains expected content': (r) => r.body.includes('voucher not found'), 
    });
}

function testVoucherFound() {
    const payload = JSON.stringify({
        query: "query { voucher(voucherIndex: 1, inputIndex: 1) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testVoucherFound is status 200': (r) => r.status === 200,
        'testVoucherFound response body contains expected content': (r) => r.body.includes('{"data":{"voucher":{"index":1}}}'), 
    });
}


function testNoticeFound() {
    const payload = JSON.stringify({
        query: "query { notice(noticeIndex: 1, inputIndex: 1) { index payload }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testNoticeFound is status 200': (r) => r.status === 200,
        'testNoticeFound response body contains expected content': (r) => r.body.includes('{"data":{"notice":{"index":1,"payload":"OX1223"}}}'), 
    });
}


function testInputFound() {
    const payload = JSON.stringify({
        query: "query { input(index: 1) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testInputFound is status 200': (r) => r.status === 200,
        'testInputFound response body contains expected content': (r) => r.body.includes('{"data":{"input":{"index":1}}}'), 
    });
}

function testGetInput() {
    const payload = JSON.stringify({
        query: "query { input(index: 1) { index }}"
    }); 

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testInputFound is status 200': (r) => r.status === 200,
        'testInputFound response body contains expected content': (r) => r.body.includes('{"data":{"input":{"index":1}}}'), 
    });
}

function testConvenientVouchers() {
    const payload = JSON.stringify({
        query: "query { convenientVouchers(first: 10) { edges { node { index }}}}"
    }); 

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testConvenientVouchers is status 200': (r) => r.status === 200,
        'testConvenientVouchers response body contains expected content': (r) => r.body.includes('{"data":{"convenientVouchers":{"edges":[{"node":{"index":1}},{"node":{"index":2}}]}}}'), 
    });
}
}

export default function () {
   testVoucherNotFound()
   testVoucherFound()
   testNoticeFound()
   testInputFound()
   testGetInput()
   testConvenientVouchers()
}
