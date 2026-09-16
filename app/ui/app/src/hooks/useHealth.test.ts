import { describe, expect, it } from "vitest";
import { healthRefetchInterval } from "./useHealth";

describe("healthRefetchInterval", () => {
  it("keeps polling when the first request has not produced data", () => {
    expect(healthRefetchInterval({ state: { data: undefined } })).toBe(1000);
  });

  it("keeps polling while the server is unhealthy", () => {
    expect(healthRefetchInterval({ state: { data: false } })).toBe(1000);
  });

  it("slows polling after the server is healthy so later crashes still self-heal", () => {
    expect(healthRefetchInterval({ state: { data: true } })).toBe(5000);
  });
});
