<script lang="ts">
  import { onMount } from 'svelte';
  import { authStatus } from './lib/stores';
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
  }
</style>
