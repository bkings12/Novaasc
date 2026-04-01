<script lang="ts">
  import { login } from '$lib/auth';
  import { goto } from '$app/navigation';

  let tenant = 'default';
  let email = 'admin@novaacs.local';
  let password = '';
  let error = '';
  let loading = false;

  async function handleLogin() {
    loading = true;
    error = '';
    try {
      await login(tenant, email, password);
      goto('/');
    } catch {
      error = 'Invalid credentials';
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-gray-950 flex items-center justify-center">
  <div class="bg-gray-900 border border-gray-800 rounded-xl p-8 w-full max-w-md">
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-white">NovaACS</h1>
      <p class="text-gray-400 text-sm mt-1">Auto Configuration Server</p>
    </div>

    {#if error}
      <div
        class="bg-red-900/30 border border-red-700 text-red-400 rounded-lg p-3 mb-4 text-sm"
      >
        {error}
      </div>
    {/if}

    <form on:submit|preventDefault={handleLogin} class="space-y-4">
      <div>
        <label for="tenant" class="text-xs text-gray-400 uppercase tracking-wider">
          Tenant
        </label>
        <input
          id="tenant"
          bind:value={tenant}
          class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm mt-1 focus:outline-none focus:border-blue-500"
        />
      </div>
      <div>
        <label for="email" class="text-xs text-gray-400 uppercase tracking-wider">
          Email
        </label>
        <input
          id="email"
          type="email"
          bind:value={email}
          class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm mt-1 focus:outline-none focus:border-blue-500"
        />
      </div>
      <div>
        <label for="password" class="text-xs text-gray-400 uppercase tracking-wider">
          Password
        </label>
        <input
          id="password"
          type="password"
          bind:value={password}
          class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-sm mt-1 focus:outline-none focus:border-blue-500"
        />
      </div>
      <button
        type="submit"
        disabled={loading}
        class="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg py-2 text-sm font-medium transition-colors"
      >
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  </div>
</div>
