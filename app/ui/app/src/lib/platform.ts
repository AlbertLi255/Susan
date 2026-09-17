export function isWindowsPlatform(): boolean {
  if (typeof window !== "undefined" && window.SUSAN_PLATFORM) {
    return window.SUSAN_PLATFORM === "windows";
  }

  return (
    typeof navigator !== "undefined" &&
    navigator.platform.toLowerCase().includes("win")
  );
}
