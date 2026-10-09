<template>
  <div class="space-y-3 animate-pulse">
    <div v-for="i in 5" :key="i" class="bg-slate-900 border border-slate-800 p-4 rounded-2xl flex items-center justify-between gap-4">
      <div class="flex items-start gap-3 w-2/3">
        <div class="w-5 h-5 bg-slate-800 rounded mt-1"></div>
        <div class="space-y-2.5 w-full">
          <div class="flex items-center gap-2">
            <div class="h-3 w-24 bg-slate-800 rounded"></div>
            <div class="h-4 w-16 bg-slate-800 rounded-full"></div>
            <div class="h-3 w-16 bg-slate-800 rounded"></div>
          </div>
          <div class="h-4 w-4/5 bg-slate-800/80 rounded"></div>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <div class="h-8 w-24 bg-slate-800 rounded-xl"></div>
        <div class="h-7 w-12 bg-slate-800 rounded-lg"></div>
        <div class="h-7 w-12 bg-slate-800 rounded-lg"></div>
      </div>
    </div>
  </div>
</template>
