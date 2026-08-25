import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

/** jsdom 30 ships no Web Storage implementation, and Node's own experimental
 *  `localStorage` global is undefined unless the process was started with
 *  --localstorage-file. EstatesPage reads its bearer token from localStorage on
 *  mount, so without this the component throws for a reason that has nothing to
 *  do with the component.
 *
 *  Deliberately a real object with real methods rather than a bag of vi.fn()s:
 *  the "never parks the secret in localStorage" assertion spies on THIS object,
 *  and a spy on a stub that was already a spy proves nothing. */
class MemoryStorage {
  private entries = new Map<string, string>();

  get length() {
    return this.entries.size;
  }
  key(index: number): string | null {
    return Array.from(this.entries.keys())[index] ?? null;
  }
  getItem(key: string): string | null {
    const v = this.entries.get(String(key));
    return v === undefined ? null : v;
  }
  setItem(key: string, value: string): void {
    this.entries.set(String(key), String(value));
  }
  removeItem(key: string): void {
    this.entries.delete(String(key));
  }
  clear(): void {
    this.entries.clear();
  }
}

function installStorage(name: 'localStorage' | 'sessionStorage') {
  const store = new MemoryStorage();
  for (const target of [globalThis, window] as unknown as Record<string, unknown>[]) {
    Object.defineProperty(target, name, { value: store, configurable: true, writable: true });
  }
}

if (!window.localStorage) installStorage('localStorage');
if (!window.sessionStorage) installStorage('sessionStorage');

// jsdom implements neither of these and the headless UI primitives behind the
// shadcn Select call both on mount.
if (!('ResizeObserver' in globalThis)) {
  (globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

// jsdom implements no layout, so it ships no scrollIntoView. TaskLogs pins the
// pane to its newest line with one on every append; without this a test that
// merely QUEUES a task fails inside an effect, far from what it was asserting.
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {};
}

if (!window.matchMedia) {
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia;
}

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});
