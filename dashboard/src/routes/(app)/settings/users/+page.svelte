<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentUser } from '$lib/auth';
  import type { ApiUser } from '$lib/types';

  let users: ApiUser[] = [];
  let loading = true;
  let msg = '';
  let err = '';

  let newEmail = '';
  let newPassword = '';
  let newRole: 'admin' | 'viewer' = 'viewer';

  function displayRole(r: string) {
    return r === 'readonly' ? 'viewer' : r;
  }

  async function load() {
    loading = true;
    err = '';
    try {
      const res = await api.users.list();
      users = res.data ?? [];
    } catch (e) {
      err = (e as Error).message;
      users = [];
    } finally {
      loading = false;
    }
  }

  onMount(load);

  async function createUser() {
    msg = '';
    err = '';
    if (!newEmail.trim() || newPassword.length < 8) {
      err = 'Email and password (min 8 characters) required';
      return;
    }
    try {
      await api.users.create({
        email: newEmail.trim(),
        password: newPassword,
        role: newRole === 'viewer' ? 'viewer' : 'admin'
      });
      newEmail = '';
      newPassword = '';
      newRole = 'viewer';
      msg = 'User created';
      await load();
    } catch (e) {
      err = (e as Error).message;
    }
  }

  async function removeUser(u: ApiUser) {
    if (!$currentUser?.user_id || u.id === $currentUser.user_id) return;
    if (!confirm(`Deactivate ${u.email}? They will no longer be able to sign in.`)) return;
    err = '';
    msg = '';
    try {
      await api.users.delete(u.id);
      msg = 'User deactivated';
      await load();
    } catch (e) {
      err = (e as Error).message;
    }
  }
</script>

<div class="p-6 max-w-3xl">
  <h1 class="text-xl font-semibold mb-2">Users</h1>
  <p class="text-sm text-gray-500 mb-6">Manage users for your tenant (admin only).</p>

  {#if msg}
    <div class="bg-green-900/20 border border-green-800 text-green-400 rounded-lg p-3 text-sm mb-4">{msg}</div>
  {/if}
  {#if err}
    <div class="bg-red-900/20 border border-red-800 text-red-400 rounded-lg p-3 text-sm mb-4">{err}</div>
  {/if}

  <div class="bg-gray-900 border border-gray-800 rounded-xl p-4 mb-8">
    <h2 class="text-xs text-gray-500 uppercase tracking-wider mb-3">New user</h2>
    <div class="grid gap-3 sm:grid-cols-2">
      <div>
        <label class="text-xs text-gray-400" for="em">Email</label>
        <input
          id="em"
          type="email"
          bind:value={newEmail}
          class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white"
        />
      </div>
      <div>
        <label class="text-xs text-gray-400" for="pw">Password</label>
        <input
          id="pw"
          type="password"
          bind:value={newPassword}
          autocomplete="new-password"
          class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white"
        />
      </div>
      <div>
        <label class="text-xs text-gray-400" for="role">Role</label>
        <select
          id="role"
          bind:value={newRole}
          class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white"
        >
          <option value="viewer">Viewer (read-only)</option>
          <option value="admin">Admin</option>
        </select>
      </div>
      <div class="flex items-end">
        <button
          type="button"
          on:click={createUser}
          class="w-full sm:w-auto px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
        >
          Create user
        </button>
      </div>
    </div>
  </div>

  {#if loading}
    <p class="text-gray-500">Loading…</p>
  {:else}
    <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800 text-gray-400 text-xs uppercase">
            <th class="px-4 py-3 text-left">Email</th>
            <th class="px-4 py-3 text-left">Role</th>
            <th class="px-4 py-3 text-left">Active</th>
            <th class="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-800">
          {#each users as u}
            <tr>
              <td class="px-4 py-2 text-white">{u.email}</td>
              <td class="px-4 py-2 text-gray-400">{displayRole(u.role)}</td>
              <td class="px-4 py-2">{u.active ? 'Yes' : 'No'}</td>
              <td class="px-4 py-2 text-right">
                {#if u.id !== $currentUser?.user_id && u.active}
                  <button
                    type="button"
                    on:click={() => removeUser(u)}
                    class="text-red-400 hover:text-red-300 text-xs"
                  >
                    Deactivate
                  </button>
                {:else if u.id === $currentUser?.user_id}
                  <span class="text-gray-600 text-xs">You</span>
                {/if}
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="4" class="px-4 py-8 text-center text-gray-500">No users</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
