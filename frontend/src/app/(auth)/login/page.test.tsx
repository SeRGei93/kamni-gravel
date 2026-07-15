// @vitest-environment jsdom

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import LoginPage from './page';

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  push: vi.fn(),
}));

vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: mocks.push }),
}));

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ login: mocks.login }),
}));

vi.mock('@/icons', () => ({
  EyeCloseIcon: () => null,
  EyeIcon: () => null,
}));

describe('LoginPage', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    mocks.login.mockReset().mockResolvedValue(undefined);
    mocks.push.mockReset();
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    (
      globalThis as typeof globalThis & {
        IS_REACT_ACT_ENVIRONMENT: boolean;
      }
    ).IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.restoreAllMocks();
  });

  it('submits credentials without writing them to the console', async () => {
    const consoleLog = vi.spyOn(console, 'log').mockImplementation(() => undefined);

    await act(async () => root.render(<LoginPage />));

    const usernameInput = container.querySelector<HTMLInputElement>(
      'input[name="username"]'
    );
    const passwordInput = container.querySelector<HTMLInputElement>(
      'input[name="password"]'
    );
    const form = container.querySelector('form');

    expect(usernameInput).not.toBeNull();
    expect(passwordInput).not.toBeNull();
    expect(form).not.toBeNull();

    usernameInput!.value = 'qa-admin';
    passwordInput!.value = 'test-password';

    await act(async () => {
      form!.dispatchEvent(
        new SubmitEvent('submit', { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    expect(mocks.login).toHaveBeenCalledWith({
      username: 'qa-admin',
      password: 'test-password',
    });
    expect(mocks.push).toHaveBeenCalledWith('/');
    expect(consoleLog).not.toHaveBeenCalled();
  });
});
