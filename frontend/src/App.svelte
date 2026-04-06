<script lang="ts">
  import { onMount } from 'svelte';
  import { authStatus, errorMessage } from './lib/stores';
  import { GetAuthStatus } from './lib/api';
  import AuthFlow from './components/AuthFlow.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import TopBar from './components/TopBar.svelte';
  import FileGrid from './components/FileGrid.svelte';
  import UploadZone from './components/UploadZone.svelte';
  import UploadProgress from './components/UploadProgress.svelte';

  onMount(async () => {
    try {
      const status = await GetAuthStatus();
      authStatus.set(status);
    } catch (e) {
      console.error('auth check:', e);
    }
  });
</script>

{#if !$authStatus.authenticated}
  <AuthFlow />
{:else}
  <div class="app-layout">
    <Sidebar />
    <div class="main-content">
      <TopBar />
      <FileGrid />
    </div>
  </div>
  <UploadZone />
  <UploadProgress />
  {#if $errorMessage}
    <div class="error-toast" on:click={() => errorMessage.set(null)}>
      {$errorMessage}
    </div>
  {/if}
{/if}

<style>
  .app-layout {
    display: flex;
    height: 100%;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    --wails-drop-target: drop;
  }

  .error-toast {
    position: fixed;
    bottom: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: #ef4444;
    color: white;
    padding: 12px 24px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    z-index: 400;
    cursor: pointer;
    box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    max-width: 80vw;
    text-align: center;
  }
</style>
