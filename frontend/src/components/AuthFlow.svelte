<script lang="ts">
  import { authStatus, errorMessage, isConfigured } from '../lib/stores';
  import { SendAuthCode, VerifyAuthCode, VerifyPassword, GetAuthStatus, IsConfigured, SaveConfig } from '../lib/api';

  let phone = '';
  let code = '';
  let password = '';
  let step: 'config' | 'phone' | 'code' | 'password' = 'phone';
  let loading = false;
  let apiId = '';
  let apiHash = '';

  IsConfigured().then(configured => {
    if (!configured) {
      step = 'config';
      isConfigured.set(false);
    }
  });

  async function saveApiConfig() {
    loading = true;
    errorMessage.set(null);
    try {
      await SaveConfig(parseInt(apiId), apiHash);
      isConfigured.set(true);
      step = 'phone';
      // Need to restart app after config change
      errorMessage.set('Config saved. Please restart the app.');
    } catch (e: any) {
      errorMessage.set(e.toString());
    } finally {
      loading = false;
    }
  }

  async function sendCode() {
    loading = true;
    errorMessage.set(null);
    try {
      await SendAuthCode(phone);
      step = 'code';
    } catch (e: any) {
      errorMessage.set(e.toString());
    } finally {
      loading = false;
    }
  }

  async function verifyCode() {
    loading = true;
    errorMessage.set(null);
    try {
      await VerifyAuthCode(code);
      const status = await GetAuthStatus();
      authStatus.set(status);
    } catch (e: any) {
      if (e.toString().includes('SESSION_PASSWORD_NEEDED')) {
        step = 'password';
        errorMessage.set(null);
      } else {
        errorMessage.set(e.toString());
      }
    } finally {
      loading = false;
    }
  }

  async function verifyPw() {
    loading = true;
    errorMessage.set(null);
    try {
      await VerifyPassword(password);
      const status = await GetAuthStatus();
      authStatus.set(status);
    } catch (e: any) {
      errorMessage.set(e.toString());
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent, fn: () => void) {
    if (e.key === 'Enter') fn();
  }
</script>

<div class="auth-container">
  <div class="auth-card">
    <h1 class="title">TG Drive</h1>
    <p class="subtitle">Telegram Cloud Storage</p>

    {#if $errorMessage}
      <div class="error">{$errorMessage}</div>
    {/if}

    {#if step === 'config'}
      <div class="step">
        <label>Telegram API ID</label>
        <input type="text" bind:value={apiId} placeholder="12345678" on:keydown={(e) => handleKeydown(e, saveApiConfig)} />
        <label>Telegram API Hash</label>
        <input type="text" bind:value={apiHash} placeholder="your_api_hash" on:keydown={(e) => handleKeydown(e, saveApiConfig)} />
        <button on:click={saveApiConfig} disabled={loading || !apiId || !apiHash}>
          {loading ? 'Saving...' : 'Save Config'}
        </button>
        <p class="hint">Get credentials from my.telegram.org</p>
      </div>
    {:else if step === 'phone'}
      <div class="step">
        <label>Phone Number</label>
        <input type="tel" bind:value={phone} placeholder="+821012345678" on:keydown={(e) => handleKeydown(e, sendCode)} autofocus />
        <button on:click={sendCode} disabled={loading || !phone}>
          {loading ? 'Sending...' : 'Send Code'}
        </button>
      </div>
    {:else if step === 'code'}
      <div class="step">
        <label>Verification Code</label>
        <input type="text" bind:value={code} placeholder="12345" on:keydown={(e) => handleKeydown(e, verifyCode)} autofocus />
        <button on:click={verifyCode} disabled={loading || !code}>
          {loading ? 'Verifying...' : 'Verify'}
        </button>
      </div>
    {:else if step === 'password'}
      <div class="step">
        <label>2FA Password</label>
        <input type="password" bind:value={password} placeholder="Password" on:keydown={(e) => handleKeydown(e, verifyPw)} autofocus />
        <button on:click={verifyPw} disabled={loading || !password}>
          {loading ? 'Verifying...' : 'Login'}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .auth-container {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    background: var(--bg-primary);
  }

  .auth-card {
    background: var(--bg-secondary);
    border-radius: 12px;
    padding: 48px;
    width: 380px;
    box-shadow: 0 8px 32px var(--shadow);
  }

  .title {
    font-size: 28px;
    font-weight: 700;
    margin-bottom: 4px;
    color: var(--accent);
  }

  .subtitle {
    color: var(--text-secondary);
    margin-bottom: 32px;
  }

  .step {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  label {
    color: var(--text-secondary);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  input {
    padding: 12px 16px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    border-radius: 8px;
    color: var(--text-primary);
    font-size: 16px;
  }

  input:focus {
    border-color: var(--accent);
  }

  button {
    padding: 12px;
    background: var(--accent);
    color: white;
    border-radius: 8px;
    font-weight: 600;
    margin-top: 8px;
    transition: background 0.2s;
  }

  button:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .error {
    background: rgba(255, 77, 106, 0.15);
    color: var(--danger);
    padding: 10px 14px;
    border-radius: 8px;
    margin-bottom: 16px;
    font-size: 13px;
  }

  .hint {
    color: var(--text-secondary);
    font-size: 12px;
    text-align: center;
  }
</style>
