<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import type { ProvisioningRule, RuleAction } from '$lib/types';

  let rules: ProvisioningRule[] = [];
  let loading = true;
  let err = '';
  let msg = '';

  let showModal = false;
  let formName = '';
  let formTrigger = 'ANY';
  let formOui = '';
  let formModel = '';
  let formPriority = 10;
  let formActionType = 'SetParameterValues';
  let formActionPriority = 5;
  /** JSON object for SetParameterValues, or JSON array of strings for GetParameterValues */
  let formActionParams = '{\n  \n}';
  let saving = false;

  const triggers = ['ANY', '0 BOOTSTRAP', '1 BOOT', '2 PERIODIC'] as const;
  const actionTypes = [
    'SetParameterValues',
    'GetParameterValues',
    'Reboot',
    'FactoryReset',
    'GetParameterNames',
    'ScheduleInform'
  ] as const;

  async function load() {
    loading = true;
    err = '';
    try {
      const res = await api.provisioning.list();
      rules = res.data ?? [];
    } catch (e) {
      err = (e as Error).message;
      rules = [];
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function openModal() {
    showModal = true;
    formName = '';
    formTrigger = 'ANY';
    formOui = '';
    formModel = '';
    formPriority = 10;
    formActionType = 'SetParameterValues';
    formActionPriority = 5;
    formActionParams = '{\n  \n}';
    err = '';
    msg = '';
  }

  function buildActions(): RuleAction[] {
    const t = formActionType;
    if (
      t === 'Reboot' ||
      t === 'FactoryReset' ||
      t === 'ScheduleInform'
    ) {
      return [{ type: t, priority: formActionPriority }];
    }
    const raw = formActionParams.trim();
    if (t === 'GetParameterValues' || t === 'GetParameterNames') {
      const names = raw ? (JSON.parse(raw) as string[]) : [];
      if (!Array.isArray(names)) throw new Error('Params must be a JSON array of parameter paths');
      return [
        {
          type: t,
          parameter_names: names,
          priority: formActionPriority
        }
      ];
    }
    const obj = raw ? (JSON.parse(raw) as Record<string, string>) : {};
    if (typeof obj !== 'object' || obj === null || Array.isArray(obj))
      throw new Error('Params must be a JSON object of path → value');
    return [
      {
        type: 'SetParameterValues',
        parameter_values: obj,
        priority: formActionPriority
      }
    ];
  }

  async function saveRule() {
    err = '';
    saving = true;
    try {
      const actions = buildActions();
      await api.provisioning.create({
        name: formName.trim(),
        trigger: formTrigger,
        match_oui: formOui.trim(),
        match_model_name: formModel.trim(),
        priority: formPriority,
        actions
      });
      msg = 'Rule created';
      showModal = false;
      await load();
    } catch (e) {
      err = (e as Error).message;
    } finally {
      saving = false;
    }
  }

  async function deleteRule(r: ProvisioningRule) {
    if (!confirm(`Delete rule "${r.name}"? This cannot be undone.`)) return;
    err = '';
    msg = '';
    try {
      await api.provisioning.delete(r.id);
      msg = 'Rule deleted';
      await load();
    } catch (e) {
      err = (e as Error).message;
    }
  }

  function actionsPreview(r: ProvisioningRule): string {
    if (!r.actions?.length) return '—';
    return r.actions
      .map((a) => a.type)
      .filter(Boolean)
      .join(', ');
  }
</script>

<div class="p-6">
  <div class="flex items-center justify-between mb-6 gap-4 flex-wrap">
    <h1 class="text-xl font-semibold">Provisioning rules</h1>
    <button
      type="button"
      on:click={openModal}
      class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm"
    >
      New rule
    </button>
  </div>

  {#if msg}
    <div class="bg-green-900/20 border border-green-800 text-green-400 rounded-lg p-3 text-sm mb-4">{msg}</div>
  {/if}
  {#if err && !showModal}
    <div class="bg-red-900/20 border border-red-800 text-red-400 rounded-lg p-3 text-sm mb-4">{err}</div>
  {/if}

  {#if loading}
    <p class="text-gray-500">Loading…</p>
  {:else}
    <div class="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800 text-gray-400 text-xs uppercase">
            <th class="px-4 py-3 text-left">Name</th>
            <th class="px-4 py-3 text-left">Trigger</th>
            <th class="px-4 py-3 text-left">Match</th>
            <th class="px-4 py-3 text-left">Actions</th>
            <th class="px-4 py-3 text-left">Prio</th>
            <th class="px-4 py-3 text-left">Active</th>
            <th class="px-4 py-3 text-right"> </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-800">
          {#each rules as r}
            <tr class="hover:bg-gray-800/50">
              <td class="px-4 py-3 text-gray-200">{r.name}</td>
              <td class="px-4 py-3 text-gray-400">{r.trigger}</td>
              <td class="px-4 py-3 text-gray-500 text-xs font-mono">
                {#if r.match_oui}OUI {r.match_oui}<br />{/if}
                {#if r.match_model_name}Model {r.match_model_name}{/if}
                {#if !r.match_oui && !r.match_model_name}—{/if}
              </td>
              <td class="px-4 py-3 text-gray-400 text-xs">{actionsPreview(r)}</td>
              <td class="px-4 py-3 text-gray-500">{r.priority}</td>
              <td class="px-4 py-3">{r.active ? 'Yes' : 'No'}</td>
              <td class="px-4 py-3 text-right">
                <button
                  type="button"
                  class="text-red-400 hover:text-red-300 text-xs"
                  on:click={() => deleteRule(r)}
                >
                  Delete
                </button>
              </td>
            </tr>
          {:else}
            <tr>
              <td colspan="7" class="px-4 py-8 text-center text-gray-500">No rules</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60"
    role="presentation"
    on:click|self={() => (showModal = false)}
  >
    <div
      class="bg-gray-900 border border-gray-800 rounded-xl max-w-lg w-full max-h-[90vh] overflow-y-auto p-6 shadow-xl"
      role="dialog"
      aria-modal="true"
      aria-labelledby="prov-modal-title"
    >
      <h2 id="prov-modal-title" class="text-lg font-semibold mb-4">New provisioning rule</h2>

      {#if err}
        <div class="bg-red-900/20 border border-red-800 text-red-400 rounded-lg p-3 text-sm mb-4">{err}</div>
      {/if}

      <div class="space-y-3 text-sm">
        <div>
          <label class="text-xs text-gray-400" for="pn">Name</label>
          <input
            id="pn"
            bind:value={formName}
            class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>
        <div>
          <label class="text-xs text-gray-400" for="pt">Inform event (trigger)</label>
          <select
            id="pt"
            bind:value={formTrigger}
            class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"
          >
            {#each triggers as tr}
              <option value={tr}>{tr}</option>
            {/each}
          </select>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="text-xs text-gray-400" for="poui">Match OUI (optional)</label>
            <input
              id="poui"
              bind:value={formOui}
              placeholder="e.g. 48A493"
              class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white font-mono text-xs"
            />
          </div>
          <div>
            <label class="text-xs text-gray-400" for="pmodel">Match model (optional)</label>
            <input
              id="pmodel"
              bind:value={formModel}
              class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white text-xs"
            />
          </div>
        </div>
        <div>
          <label class="text-xs text-gray-400" for="pprio">Rule priority</label>
          <input
            id="pprio"
            type="number"
            bind:value={formPriority}
            class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>
        <div>
          <label class="text-xs text-gray-400" for="pat">Action (task type)</label>
          <select
            id="pat"
            bind:value={formActionType}
            class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"
          >
            {#each actionTypes as at}
              <option value={at}>{at}</option>
            {/each}
          </select>
        </div>
        <div>
          <label class="text-xs text-gray-400" for="pap">Action priority</label>
          <input
            id="pap"
            type="number"
            bind:value={formActionPriority}
            class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white"
          />
        </div>
        {#if formActionType === 'SetParameterValues'}
          <div>
            <label class="text-xs text-gray-400" for="pjson">Parameter values (JSON object)</label>
            <textarea
              id="pjson"
              bind:value={formActionParams}
              rows="6"
              class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white font-mono text-xs"
              placeholder={'{\n  "Device.Foo.Bar": "value"\n}'}
            ></textarea>
          </div>
        {:else if formActionType === 'GetParameterValues' || formActionType === 'GetParameterNames'}
          <div>
            <label class="text-xs text-gray-400" for="pjson2">Parameter paths (JSON array)</label>
            <textarea
              id="pjson2"
              bind:value={formActionParams}
              rows="5"
              class="w-full mt-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white font-mono text-xs"
              placeholder={'["Device."]'}
            ></textarea>
          </div>
        {:else}
          <p class="text-xs text-gray-500">No extra parameters for this action type.</p>
        {/if}
      </div>

      <div class="flex justify-end gap-2 mt-6">
        <button
          type="button"
          class="px-4 py-2 text-gray-400 hover:text-white text-sm"
          on:click={() => (showModal = false)}
        >
          Cancel
        </button>
        <button
          type="button"
          disabled={saving || !formName.trim()}
          class="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-40 text-white rounded-lg text-sm"
          on:click={saveRule}
        >
          {saving ? 'Saving…' : 'Create'}
        </button>
      </div>
    </div>
  </div>
{/if}
