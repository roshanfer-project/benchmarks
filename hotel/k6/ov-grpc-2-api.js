import { Client, StatusOK, StatusResourceExhausted } from 'k6/net/grpc';
import { Counter, Trend, Rate, Gauge } from 'k6/metrics';
import { getCurrentStageIndex } from 'https://jslib.k6.io/k6-utils/1.3.0/index.js';

const successLatency = new Trend('success_req_latency', true);
const successCounter  = new Counter('success_counter');
const fail_503_latency = new Trend('fail_503_latency', true);
const fail503Counter = new Counter('fail_503_counter');
const failOtherCounter = new Counter('fail_other_counter');
const offeredLoad = new Gauge('offered_load');
const requestRate = new Rate('request_rate');


// Create one client per VU
const client = new Client();
let isConnected = false;

export const options = {
  discardResponseBodies: true,
  scenarios: {
    contacts: {
      executor: 'ramping-arrival-rate',

      startRate: 1200,

      stages: [
        { target: 1200, duration: '5s' },
        { target: 5000, duration: '0s' },
        { target: 5000, duration: '10s' }
      ],

      // Start `rate` iterations per second
      timeUnit: '1s',

      // Pre-allocate 2 VUs before starting the test
      preAllocatedVUs: 50,

      // Spin up a maximum of 50 VUs to sustain the defined
      // constant arrival rate.
      maxVUs: 500,
    },
  },
};


export function setup() {
  // This runs once globally, not per VU
  return {};
}

export default function (data) {
  // Connect only once per VU
  if (!isConnected) {
    client.connect('192.168.1.100:3000', { reflect: true, plaintext: true });
    isConnected = true;
  }
  
  const startTime = Date.now();
  if (getCurrentStageIndex() == 0) {
    var res = client.invoke('protobuf.RajomonClient/SearchHotels', {
        lat: 37.7867,
        lon: -122.4112,
        InDate: '2024-08-15',
        OutDate: '2024-08-17',
      });
      var api = 'SearchHotels';
  } else {
    const randomNumber = Math.random();
    if (randomNumber <= 0.40) {
      var res = client.invoke('protobuf.RajomonClient/SearchHotels', {
        lat: 37.7867,
        lon: -122.4112,
        InDate: '2024-08-15',
        OutDate: '2024-08-17',
      });
      var api = 'SearchHotels';
    } else {
      var res = client.invoke('protobuf.RajomonClient/FrontendReservation', {
        HotelId: "4",
        CustomerName: "Alice",
        Username: "Cornell_1",
        Password: "1111111111",
        Number: 1,
        InDate: "2025-05-20",
        OutDate: "2025-05-22",
      });
      var api = 'FrontendReservation';
    }
  }
  
  const duration = Date.now() - startTime;

  requestRate.add(1, { api_name: api });

  if (res.status === StatusOK) {
    successCounter.add(1, { api_name: api });
    successLatency.add(duration, { api_name: api });
  } else if (res.status === StatusResourceExhausted) {
    //console.info(`message: ${res.error.message}`);
    fail503Counter.add(1, { api_name: api });
    fail_503_latency.add(duration, { api_name: api });
  } 
  else {
    failOtherCounter.add(1, { api_name: api });
    // print the status code
    console.error(
      `Unexpected status code: ${res.status}, message: ${res.error.message}`
    );
  }
  
  // Don't close the connection - reuse it for the next iteration
}

export function teardown(data) {
  // Clean up the connection when the test ends
  if (isConnected) {
    client.close();
  }
}