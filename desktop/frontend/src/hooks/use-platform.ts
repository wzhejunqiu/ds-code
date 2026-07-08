import { useEffect } from "react";
import { System } from "@wailsio/runtime";

/** Sets `data-platform` on `<html>` for platform-specific CSS (title bar insets, etc.). */
export function usePlatform() {
  useEffect(() => {
    const root = document.documentElement;
    if (System.IsMac()) {
      root.dataset.platform = "darwin";
    } else if (System.IsWindows()) {
      root.dataset.platform = "windows";
    } else {
      delete root.dataset.platform;
    }
  }, []);
}
