// Wails runtime types
export function EventsOn(
  eventName: string,
  callback: (...data: unknown[]) => void
): () => void;
export function EventsOff(eventName: string): void;
export function EventsEmit(eventName: string, ...data: unknown[]): void;
export function WindowSetTitle(title: string): void;
export function WindowMinimise(): void;
export function WindowMaximise(): void;
export function WindowClose(): void;
