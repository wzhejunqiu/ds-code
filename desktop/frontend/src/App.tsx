import { AppShell } from "@/app/AppShell";
import { AppProvider } from "@/state/app-store";

export default function App() {
  return (
    <AppProvider>
      <AppShell />
    </AppProvider>
  );
}
