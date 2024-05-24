import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '30s', target: 10 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        checks: ['rate>0.9'],
    }
};

export default function () {
    const url = 'http://localhost:8080/graphql'; 
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
        'is status 200': (r) => r.status === 200,
        'response body contains expected content': (r) => r.body.includes('blabla not found'), 
    });
}
