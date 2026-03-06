<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";

  let token = $derived($page.params.token);

  onMount(() => {
    goto(`/convite/${token}`, { replaceState: true });
  });
</script>
