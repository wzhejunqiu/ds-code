import { AppShell } from "@/app/AppShell";
import { usePlatform } from "@/hooks/use-platform";
import { AppProvider } from "@/state/app-store";

export default function App() {
  usePlatform();
  return (
    <AppProvider>
      <AppShell />
    </AppProvider>
  );
}
