# Mock layer patterns

Concrete shapes for the mock data module described in Step 2 of `SKILL.md`. Adapt naming to the
project; keep the structure — one module, async accessors, in-memory writes, a failure switch.

## Plain JS (ES modules)

`js/orders.mock.js`

```js
// =====================================================================
// MOCK DATA — nothing in this file talks to a server.
//
// REPLACING THE MOCKS
//   getOrders(status)  -> GET    /api/orders?status=<status>   -> Order[]
//   getOrder(id)       -> GET    /api/orders/<id>              -> Order
//   updateOrder(id, p) -> PATCH  /api/orders/<id>   body: p    -> Order
//   deleteOrder(id)    -> DELETE /api/orders/<id>              -> 204
// Keep the Order shape below as the contract. Delete this file once the
// endpoints exist — only ./orders-view.js imports it.
// =====================================================================

/** Flip these from the console or a dev control to reach the other states. */
export const MOCK_CONFIG = { failNext: false, emptyMode: false, delayMs: 400 };

// MOCK: hardcoded session. Replace with the real auth/session context.
export const MOCK_CURRENT_USER = { id: 'u_1', name: 'Ana Petrović', role: 'admin' };

/** @typedef {{id: string, customer: string, total: number, status: 'open'|'shipped'|'cancelled', placedAt: string, note: string|null}} Order */

// Realistic spread: long + short names, a null, both extremes of total.
let orders = [
  { id: 'ord_1042', customer: 'Wilhelmina Vandersteen-Ashworth', total: 12480.0, status: 'open',      placedAt: '2026-08-14T09:12:00Z', note: 'Deliver to loading bay 3, ask for Marko.' },
  { id: 'ord_1041', customer: 'Li Wei',                          total: 12.5,    status: 'shipped',   placedAt: '2026-08-13T17:44:00Z', note: null },
  { id: 'ord_1040', customer: 'Ítalo Gonçalves',                 total: 349.99,  status: 'cancelled', placedAt: '2026-08-11T08:03:00Z', note: 'Customer changed their mind.' },
  // ...at least 8-12 rows so pagination and scrolling get exercised
];

// MOCK: artificial latency so loading states are real. Delete with the file.
const settle = (value) =>
  new Promise((resolve, reject) =>
    setTimeout(() => {
      if (MOCK_CONFIG.failNext) {
        MOCK_CONFIG.failNext = false;
        reject(new Error('Mock failure: the request could not be completed.'));
      } else {
        resolve(value);
      }
    }, MOCK_CONFIG.delayMs),
  );

// MOCK: replace with GET /api/orders?status=<status>
export function getOrders(status = 'all') {
  if (MOCK_CONFIG.emptyMode) return settle([]);
  const rows = status === 'all' ? orders : orders.filter((o) => o.status === status);
  return settle(structuredClone(rows));
}

// MOCK: replace with GET /api/orders/<id>
export function getOrder(id) {
  const found = orders.find((o) => o.id === id);
  return found ? settle(structuredClone(found)) : settle(Promise.reject(new Error('Not found')));
}

// MOCK: replace with PATCH /api/orders/<id>. Writes are in memory only — a reload resets them.
export function updateOrder(id, patch) {
  orders = orders.map((o) => (o.id === id ? { ...o, ...patch } : o));
  return settle(structuredClone(orders.find((o) => o.id === id)));
}

// MOCK: replace with DELETE /api/orders/<id>
export function deleteOrder(id) {
  orders = orders.filter((o) => o.id !== id);
  return settle(true);
}
```

Consuming it, with the three states actually rendered:

```js
import { getOrders } from './orders.mock.js';

async function render(status) {
  setState('loading');                       // skeleton rows
  try {
    const rows = await getOrders(status);    // swap to fetch() here; the rest is unchanged
    setState(rows.length ? 'ready' : 'empty');
    paint(rows);
  } catch (err) {
    setState('error', err.message);          // retry button re-calls render(status)
  }
}
```

## React

`src/mocks/orders.mock.ts` holds the same module. A hook keeps components clean and gives one
place to swap in React Query / SWR / `fetch` later:

```tsx
// MOCK: this hook reads from the in-memory mock module.
// Replace the getOrders() call with the real request; the returned shape stays the same.
export function useOrders(status: OrderStatus | 'all') {
  const [state, setState] = useState<{ status: 'loading' | 'ready' | 'error'; data: Order[]; error?: string }>({
    status: 'loading',
    data: [],
  });

  useEffect(() => {
    let cancelled = false;
    setState((s) => ({ ...s, status: 'loading' }));
    getOrders(status)
      .then((data) => !cancelled && setState({ status: 'ready', data }))
      .catch((e) => !cancelled && setState({ status: 'error', data: [], error: e.message }));
    return () => { cancelled = true; };
  }, [status]);

  return state;
}
```

## Vue

Same module; a composable mirrors the hook:

```ts
// MOCK: backed by the in-memory mock module — swap getOrders() for the real endpoint.
export function useOrders(status: Ref<string>) {
  const data = ref<Order[]>([]);
  const status_ = ref<'loading' | 'ready' | 'error'>('loading');
  const error = ref<string | null>(null);

  watchEffect(async () => {
    status_.value = 'loading';
    try { data.value = await getOrders(status.value); status_.value = 'ready'; }
    catch (e: any) { error.value = e.message; status_.value = 'error'; }
  });

  return { data, status: status_, error };
}
```

## Modal wiring (vanilla — the pattern to match in any stack)

```js
let lastFocused = null;

function openModal(dialog) {
  lastFocused = document.activeElement;
  dialog.hidden = false;
  document.body.style.overflow = 'hidden';
  dialog.querySelector('[autofocus], button, [href], input, select, textarea')?.focus();
  document.addEventListener('keydown', onKeydown);
}

function closeModal(dialog) {
  dialog.hidden = true;
  document.body.style.overflow = '';
  document.removeEventListener('keydown', onKeydown);
  lastFocused?.focus();                       // focus returns to the trigger
}

function onKeydown(e) {
  if (e.key === 'Escape') closeModal(currentDialog);
  if (e.key === 'Tab') trapFocus(e, currentDialog);   // wrap first <-> last focusable
}
```

Markup: `<div class="modal" role="dialog" aria-modal="true" aria-labelledby="modal-title" hidden>`,
backdrop click closes, and the trigger carries `aria-haspopup="dialog"`.

Native `<dialog>` with `showModal()` gives Esc, the backdrop, and the focus trap for free — prefer
it when the project's browser targets allow.

## Dev state switcher

A small fixed-corner control makes every state reviewable without editing code. Mark it clearly so
it is deleted with the mocks:

```html
<!-- MOCK: demo-only state switcher. Delete this block when real data is wired in. -->
<fieldset class="mock-switcher">
  <legend>Mock state</legend>
  <button data-mock="ready">Ready</button>
  <button data-mock="empty">Empty</button>
  <button data-mock="error">Error</button>
  <button data-mock="slow">Slow (3s)</button>
</fieldset>
```
