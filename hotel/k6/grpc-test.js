import { Client, StatusOK, StatusResourceExhausted } from 'k6/net/grpc';
import { Counter, Trend, Rate } from 'k6/metrics';

const successLatency = new Trend('success_req_latency', true);
const successCounter  = new Counter('success_counter');
const fail_503_latency = new Trend('fail_503_latency', true);
const fail503Counter = new Counter('fail_503_counter');
const failOtherCounter = new Counter('fail_other_counter');

// Create one client per VU
const client = new Client();
let isConnected = false;

export const options = {
  discardResponseBodies: true,
  scenarios: {
    contacts: {
      executor: 'constant-arrival-rate',

      // How long the test lasts
      duration: '10s',

      // How many iterations per timeUnit
      rate: 5000,

      // Start `rate` iterations per second
      timeUnit: '1s',

      // Pre-allocate 2 VUs before starting the test
      preAllocatedVUs: 20,

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
  const res = client.invoke('protobuf.RajomonClient/FrontendReservation', {
        HotelId: "4",
        CustomerName: "Alice",
        Username: "Cornell_1",
        Password: "1111111111",
        Number: 1,
        InDate: "2025-05-20",
        OutDate: "2025-05-22",
      });
  const duration = Date.now() - startTime;


  if (res.status === StatusOK) {
    successCounter.add(1);
    successLatency.add(duration);
  } else if (res.status === StatusResourceExhausted) {
    //console.info(`message: ${res.error.message}`);
    fail503Counter.add(1);
    fail_503_latency.add(duration);
  } 
  else {
    failOtherCounter.add(1);
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