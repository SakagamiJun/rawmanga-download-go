import { describe, expect, it } from "vitest";
import { resolveLocale, resolveTheme } from "@/lib/system";

describe("resolveLocale", () => {
  it("maps zh locales to zh-CN", () => {
    expect(
      resolveLocale(
        {
          outputRoot: "",
          maxConcurrentDownloads: 6,
          retryCount: 3,
          requestTimeoutSec: 30,
          localeMode: "system",
          locale: "en",
          themeMode: "system",
        },
        ["zh-Hans-CN"]
      )
    ).toBe("zh-CN");
  });

  it("prefers manual locale when localeMode is manual", () => {
    expect(
      resolveLocale({
        outputRoot: "",
        maxConcurrentDownloads: 6,
        retryCount: 3,
        requestTimeoutSec: 30,
        localeMode: "manual",
        locale: "ja",
        themeMode: "system",
      })
    ).toBe("ja");
  });
});

describe("resolveTheme", () => {
  it("returns system dark when prefersDark is true", () => {
    expect(resolveTheme("system", true)).toBe("dark");
  });
});

