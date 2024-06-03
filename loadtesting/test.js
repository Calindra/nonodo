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

    // Log the response status and body
    console.log('Response status: ' + response.status);
    console.log('Response body: ' + response.body);

    const isStatus200 = response.status === 200;
    const isBodyContainsExpectedContent = response.body.includes('{"data":{"input":{"index":1}}}');

    check(response, {
        'testInputFound is status 200': (r) => isStatus200,
        'testInputFound response body contains expected content': (r) => isBodyContainsExpectedContent, 
    });

    // Additional logging to understand which check is failing
    if (!isStatus200) {
        console.error('Expected status 200 but got: ' + response.status);
    }
    if (!isBodyContainsExpectedContent) {
        console.error('Response body does not contain expected content.');
    }
}

export default function () {
   testVoucherNotFound()
   testVoucherFound()
   testNoticeFound()
   testInputFound()
}
