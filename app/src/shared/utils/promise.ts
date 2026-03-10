export function createPromise<T>() {
  let resolve = null as ((value: T) => void) | null;
  let reject = null as ((reason?: any) => void) | null;

  const out = {
    promise: new Promise<T>((res, rej) => {
      resolve = res;
      reject = rej;
    }),
    resolve: null as ((value: T) => void) | null,
    reject: null as ((reason?: any) => void) | null,
  };

  out.resolve = resolve;
  out.reject = reject;

  return out;
}
