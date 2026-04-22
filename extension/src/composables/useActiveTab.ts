// Watches chrome.tabs to keep a reactive `activeTab` ref in sync with the
// tab currently focused in the user's foreground window. Used by every UI
// surface so they react to tab switches without re-plumbing.
import { ref, onMounted, onBeforeUnmount } from "vue";

export interface ActiveTab {
  id: number;
  url: string;
  title: string;
}

export function useActiveTab() {
  const activeTab = ref<ActiveTab | null>(null);

  async function read() {
    const [tab] = await chrome.tabs.query({
      active: true,
      lastFocusedWindow: true,
    });
    if (!tab?.id) {
      activeTab.value = null;
      return;
    }
    activeTab.value = {
      id: tab.id,
      url: tab.url ?? "",
      title: tab.title ?? "",
    };
  }

  const onActivated = () => void read();
  const onUpdated = (
    _tabId: number,
    changeInfo: { url?: string; title?: string; status?: string },
  ) => {
    // Refresh when URL or title changes on the active tab.
    if (
      changeInfo.url ||
      changeInfo.title ||
      changeInfo.status === "complete"
    ) {
      void read();
    }
  };

  onMounted(() => {
    void read();
    chrome.tabs.onActivated.addListener(onActivated);
    chrome.tabs.onUpdated.addListener(onUpdated);
  });

  onBeforeUnmount(() => {
    chrome.tabs.onActivated.removeListener(onActivated);
    chrome.tabs.onUpdated.removeListener(onUpdated);
  });

  return { activeTab, refresh: read };
}
