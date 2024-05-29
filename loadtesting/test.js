import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '30s', target: 10 },
        { duration: '60s', target: 30 },
        { duration: '60s', target: 60 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        checks: ['rate>0.9'],
    }
};

function testVoucherNotFound() {
    const url = 'http://localhost:8181/graphql'; 
    const payload = JSON.stringify({
        query: "query { voucher(voucherIndex: 0, inputIndex: 0) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(url, payload, params);

    check(response, {
        'testVoucherNotFound is status 200': (r) => r.status === 200,
        'testVoucherNotFound response body contains expected content': (r) => r.body.includes('voucher not found'), 
    });
}

function testVoucherFound() {
    const url = 'http://localhost:8181/graphql'; 
    const payload = JSON.stringify({
        query: "query { voucher(voucherIndex: 1, inputIndex: 1) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(url, payload, params);

    check(response, {
        'testVoucherFound is status 200': (r) => r.status === 200,
        'testVoucherFound response body contains expected content': (r) => r.body.includes('{"data":{"voucher":{"index":1}}}'), 
    });
}


function testNoticeFound() {
    const url = 'http://localhost:8181/graphql'; 
    const payload = JSON.stringify({
        query: "query { notice(noticeIndex: 1, inputIndex: 1) { index payload }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(url, payload, params);

    check(response, {
        'testNoticeFound is status 200': (r) => r.status === 200,
        'testNoticeFound response body contains expected content': (r) => r.body.includes('{"data":{"notice":{"index":1,"payload":"OX1223"}}}'), 
    });
}

export default function () {
   testVoucherNotFound()
   testVoucherFound()
   testNoticeFound()
}
