import { sleep } from 'k6';
import http from 'k6/http';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const banners = JSON.parse(open(`${__ENV.INFO_FILE}`, 'r'));

export const options = {
    scenarios: {
        test: {
            executor: 'constant-arrival-rate',
            duration: '5m',
            preAllocatedVUs: 10,

            rate: __ENV.RATE_COUNT,
            timeUnit: '1s',
            maxVUs: 40,
        },
    },
    discardResponseBodies: true,
    thresholds: {
        http_req_failed: ['rate<0.0001'], // http errors should be less than 1%
        http_req_duration: ['p(99)<50'], // 99% of requests should be below 1s
    },
};

export default function () {
    // Get Token
    const token = 'admin-token';

    const randomBanner = randomItem(banners);

    const tagID = randomItem(randomBanner.tag_ids);
    
    const limit = randomIntBetween(0, 100);
    const offset = randomIntBetween(0, 100);

    // define URL and request body
    const url = `http://localhost:8080/api/v1/banner?feature_id=${randomBanner.feature_id}&tag_id=${tagID}&limit=${limit}&offset=${offset}`;
    const params = {
        headers: {
            'token': token,
        },
        tags: { name: "banner" },
    };

    // send a post request and save response as a variable
    http.get(url, params);
}
