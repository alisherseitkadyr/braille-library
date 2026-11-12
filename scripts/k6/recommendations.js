// Load test of GET /users/me/recommendations.
// setup() logs in 50 users (user1@perf.kz ... user50@perf.kz from the big seed),
// then 20 virtual users call the endpoint for 30 s, each request as a random one of them.
// Requests still running at 30 s may finish (gracefulStop), so slow modes are measured too.
//
//   k6 run scripts/k6/recommendations.js
//   k6 run -e SIZE=M -e MODE=live -e ROW_FILE=row.csv scripts/k6/recommendations.js
//
// With ROW_FILE, one CSV row "size,mode,p50,p95,avg,rps" is written (times in ms).
import http from "k6/http";
import { check } from "k6";
import { Trend } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const USERS = 50;
const DURATION_S = 30;
const setupTime = new Trend("setup_time", true);

export const options = {
  setupTimeout: "120s",
  scenarios: {
    recommendations: {
      executor: "constant-vus",
      vus: 20,
      duration: `${DURATION_S}s`,
      // Let slow requests (live mode on big data) finish instead of dropping them.
      gracefulStop: "120s",
    },
  },
  // Thresholds that always pass, only to get per-request metrics for the
  // recommendation calls (setup logins are excluded).
  thresholds: {
    "http_req_duration{name:recommendations}": ["max>=0"],
    "http_reqs{name:recommendations}": ["count>=0"],
    "http_req_failed{name:recommendations}": ["rate>=0"],
  },
  summaryTrendStats: ["avg", "med", "p(95)", "max"],
};

export function setup() {
  const started = Date.now();
  const tokens = [];
  for (let i = 1; i <= USERS; i++) {
    const res = http.post(
      `${BASE_URL}/auth/login`,
      JSON.stringify({ email: `user${i}@perf.kz`, password: "Demo12345" }),
      { headers: { "Content-Type": "application/json" }, tags: { name: "login" } },
    );
    if (res.status !== 200) {
      throw new Error(`login user${i}@perf.kz failed: ${res.status} ${res.body}`);
    }
    tokens.push(res.json("token"));
  }
  setupTime.add(Date.now() - started);
  return { tokens };
}

export default function (data) {
  const token = data.tokens[Math.floor(Math.random() * data.tokens.length)];
  const res = http.get(`${BASE_URL}/users/me/recommendations?limit=10`, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { name: "recommendations" },
    timeout: "120s",
  });
  check(res, {
    "status is 200": (r) => r.status === 200,
    "10 recommendations": (r) => r.status === 200 && r.json("items").length === 10,
  });
}

export function handleSummary(data) {
  const d = data.metrics["http_req_duration{name:recommendations}"].values;
  const count = data.metrics["http_reqs{name:recommendations}"].values.count;
  const failed = data.metrics["http_req_failed{name:recommendations}"].values.rate;
  // RPS = completed recommendation requests per second of load (the whole run
  // minus the login setup; k6's own rate would also count the setup).
  const loadSeconds = (data.state.testRunDurationMs - data.metrics.setup_time.values.max) / 1000;
  const rps = count / loadSeconds;
  const f = (x) => x.toFixed(1);

  const size = __ENV.SIZE || "-";
  const mode = __ENV.MODE || "-";
  const text =
    `size=${size} mode=${mode} requests=${count} failed=${(failed * 100).toFixed(2)}% ` +
    `p50=${f(d.med)}ms p95=${f(d["p(95)"])}ms avg=${f(d.avg)}ms max=${f(d.max)}ms rps=${f(rps)}\n`;

  const out = { stdout: text };
  if (__ENV.ROW_FILE) {
    out[__ENV.ROW_FILE] = [size, mode, f(d.med), f(d["p(95)"]), f(d.avg), f(rps)].join(",") + "\n";
  }
  return out;
}
