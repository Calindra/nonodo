import http from 'k6/http';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 10 },
        { duration: '10s', target: 0 },
    ],
};

export default function () {
    let res = http.get('https://test.k6.io');
    check(res, {
        'is status 200': (r) => r.status === 200,
    });
}
