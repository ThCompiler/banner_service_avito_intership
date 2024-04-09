import { sleep } from 'k6';
import http from 'k6/http';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const banners = JSON.parse(open(`${__ENV.INFO_FILE}`));

export const options = {
    scenarios: {
        smoke_test: {
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
    const token = 'user-token';

    const randomBanner = randomItem(banners);

    const tagId = randomItem(randomBanner.tag_ids);

    let use_last_revision = false;
    if (randomIntBetween(0, 100) < 50) {
        use_last_revision = true;
    }

    // define URL and request body
    const url = `http://localhost:8080/api/v1/user_banner?feature_id=${randomBanner.feature_id}&tag_id=${tagId}&use_last_revision=${use_last_revision}`;
    const params = {
        headers: {
            'token': token,
        },
        tags: { name: "user_banner" },
    };

    // send a post request and save response as a variable
    http.get(url, params);
}
