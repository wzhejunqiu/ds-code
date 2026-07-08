import type { SettingsSection } from "./constants";
import { SETTINGS_SECTIONS } from "./constants";

export function SettingsNav({
  active,
  onSelect,
}: {
  active: SettingsSection;
  onSelect: (section: SettingsSection) => void;
}) {
  return (
    <nav className="settings-nav">
      {SETTINGS_SECTIONS.map((s) => (
        <button
          key={s.id}
          type="button"
          className={`settings-nav-item${active === s.id ? " active" : ""}`}
          onClick={() => onSelect(s.id)}
        >
          {s.label}
        </button>
      ))}
    </nav>
  );
}
