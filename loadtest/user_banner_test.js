import { sleep } from 'k6';
import http from 'k6/http';
import { randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const banners = JSON.parse(open('./info.json'));

export const options = {
    scenarios: {
    my_scenario1: {
      executor: 'constant-arrival-rate',
      duration: '1m', // total duration
      preAllocatedVUs: 80, // to allocate runtime resources

      rate: 1000, // number of constant iterations given `timeUnit`
      timeUnit: '1s',
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

    // define URL and request body
    const url = `http://localhost:8080/api/v1/user_banner?feature_id=${randomBanner.feature_id}&tag_id=${tagId}&use_last_revision=true`;
    const params = {
        headers: {
            'token': token,
        },
    };

    // send a post request and save response as a variable
    http.get(url, params);
}
