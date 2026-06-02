// Wails runtime bridge — provided by Wails at runtime
export function EventsOn(eventName, callback) {
  return window.runtime?.EventsOn(eventName, callback) ?? (() => {});
}
export function EventsOff(eventName) {
  window.runtime?.EventsOff(eventName);
}
export function EventsEmit(eventName, ...data) {
  window.runtime?.EventsEmit(eventName, ...data);
}
export function WindowSetTitle(title) {
  window.runtime?.WindowSetTitle(title);
}
export function WindowMinimise() {
  window.runtime?.WindowMinimise();
}
export function WindowMaximise() {
  window.runtime?.WindowMaximise();
}
export function WindowClose() {
  window.runtime?.WindowClose();
}
