import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Events } from "@wailsio/runtime";

/** Routes desktop:action events and global shortcuts through React Router. */
export function DesktopActionBridge() {
  const navigate = useNavigate();

  useEffect(() => {
    const off = Events.On("desktop:action", (raw: { data: Record<string, string> }) => {
      const action = raw.data?.action;
      if (action === "open_settings") {
        navigate("/settings");
      }
    });
    return () => off();
  }, [navigate]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === ",") {
        e.preventDefault();
        navigate("/settings");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [navigate]);

  return null;
}
