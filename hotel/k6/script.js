import http from 'k6/http';
import { Counter, Trend, Rate } from 'k6/metrics';

const successLatency = new Trend('success_req_latency', true);
const successCounter  = new Counter('success_counter');
const fail_503_latency = new Trend('fail_503_latency', true);
const fail503Counter = new Counter('fail_503_counter');
const failOtherCounter = new Counter('fail_other_counter');

export const options = {
  discardResponseBodies: true,
  scenarios: {
    contacts: {
      executor: 'ramping-arrival-rate',

      startRate: 1200,

      stages: [
        { target: 1200, duration: '5s' },
        { target: 2400, duration: '0s' },
        { target: 2400, duration: '5s' }
      ],

      // Start `rate` iterations per second
      timeUnit: '1s',

      // Pre-allocate 2 VUs before starting the test
      preAllocatedVUs: 10,

      // Spin up a maximum of 50 VUs to sustain the defined
      // constant arrival rate.
      maxVUs: 500,
    },
  },
};

export default function () {
  const res = http.get('http://192.168.1.100:3000/hotels?lat=37.7867&lon=-122.4112&inDate=2024-08-15&outDate=2024-08-17');
  if (res.status === 200) {
    successLatency.add(res.timings.duration);
    successCounter.add(1);
  } else if (res.status === 503) {
    fail503Counter.add(1);
    fail_503_latency.add(res.timings.duration);
  }
  else {
    failOtherCounter.add(1);
    // print the status code
    console.error(`Unexpected status code: ${res.status}, response: ${res.body}`);
  }
}
