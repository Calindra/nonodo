import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '30s', target: 1 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        checks: ['rate>0.9'],
    }
};

const GRAPHQL_ENDPOINT = 'http://localhost:8080/graphql'

function assertStringContains(s, substring) {
    return typeof s === "string" && typeof substring === "string" && s.includes(substring);
}

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
        'testVoucherNotFound response body contains expected content': (r) => assertStringContains(r.body, 'voucher not found'),
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
        'testVoucherFound response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"voucher":{"index":1}}}'),
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
        'testNoticeFound response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"notice":{"index":1,"payload":"OX1223"}}}'),
    });
}


function testInputFound() {
    const payload = JSON.stringify({
        query: 'query { input(id: "1") { index }}'
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testInputFound is status 200': (r) => r.status === 200,
        'testInputFound response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"input":{"index":1}}}'),
    });
}

function testReportFound() {
    const payload = JSON.stringify({
        query: "query { report(reportIndex: 2, inputIndex: 2) { index }}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testReportFound is status 200': (r) => r.status === 200,
        'testReportFound response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"report":{"index":2}}}'),
    });
}

function testReports() {
    const payload = JSON.stringify({
        query: "query { reports(first: 10) { edges { node { index }}}}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testReports is status 200': (r) => r.status === 200,
        'testReports response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"reports":{"edges":[{"node":{"index":1}},{"node":{"index":2}}]}}}'),
    });
}

function testVouchers() {
    const payload = JSON.stringify({
        query: "query { vouchers(first: 10) { edges { node { index }}}}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testVouchers is status 200': (r) => r.status === 200,
        'testVouchers response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"vouchers":{"edges":[{"node":{"index":1}},{"node":{"index":2}}]}}}'),
    });
}

function testNotices() {
    const payload = JSON.stringify({
        query: "query { notices(first: 10) { edges { node { index }}}}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testNotices is status 200': (r) => r.status === 200,
        'testNotices response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"notices":{"edges":[{"node":{"index":1}},{"node":{"index":2}}]}}}'),
    });
}

function testInputs() {
    const payload = JSON.stringify({
        query: "query { inputs(first: 10) { edges { node { index }}}}"
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const response = http.post(GRAPHQL_ENDPOINT, payload, params);

    check(response, {
        'testInputs is status 200': (r) => r.status === 200,
        'testInputs response body contains expected content': (r) => assertStringContains(r.body, '{"data":{"inputs":{"edges":[{"node":{"index":1}},{"node":{"index":2}}]}}}'),
    });
}

export default function () {
    // testVoucherNotFound()
    // testVoucherFound()
    // testNoticeFound()
    testInputFound()
    // testReportFound()
    // testVouchers()
    // testNotices()
    // testReports()
    testInputs()
}
