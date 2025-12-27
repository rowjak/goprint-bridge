<script setup>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { GetPrinters, GetConfig, SaveConfig, StartServer, StopServer, IsServerRunning, PrintTestPage, MinimizeToTray, QuitApp } from '../wailsjs/goprint-bridge/appservice.js'
import { Events } from '@wailsio/runtime'

// State Management
const printers = ref([])
const selectedPrinter = ref('')
const port = ref(9999)
const autoStart = ref(false)
const isRunning = ref(false)
const statusMessage = ref('Stopped')
const isLoading = ref(false)
const isAppReady = ref(false)

// Activity Log (last 5 activities)
const activityLog = reactive([])
const MAX_LOG_ITEMS = 3

// Toast Notifications
const toasts = reactive([])
let toastId = 0
const MAX_TOASTS = 2

// Add activity to log
const addActivity = (message, type = 'info') => {
  const activity = {
    id: Date.now(),
    message,
    type, // info, success, error, print
    time: new Date()
  }
  activityLog.unshift(activity)
  if (activityLog.length > MAX_LOG_ITEMS) {
    activityLog.pop()
  }
}

// Toast functions
const showToast = (message, type = 'info', duration = 4000) => {
  const id = ++toastId
  const toast = { id, message, type, visible: true }
  toasts.push(toast)
  
  // Jika jumlah toast melebihi MAX_TOASTS, hapus toast tertua langsung
  while (toasts.length > MAX_TOASTS) {
    toasts.shift()
  }
  
  setTimeout(() => {
    const index = toasts.findIndex(t => t.id === id)
    if (index !== -1) {
      toasts[index].visible = false
      setTimeout(() => {
        const removeIndex = toasts.findIndex(t => t.id === id)
        if (removeIndex !== -1) toasts.splice(removeIndex, 1)
      }, 300)
    }
  }, duration)
}

// Last print job for display
const lastPrintJob = ref(null)

// Load printers and config on mount
onMounted(async () => {
  try {
    addActivity('Initializing GoPrintBridge...', 'info')
    
    // Load printers
    const printerList = await GetPrinters()
    printers.value = printerList || []
    addActivity(`Found ${printers.value.length} printer(s)`, 'info')
    
    // Load config
    const cfg = await GetConfig()
    if (cfg) {
      const defaultPrinter = printers.value.length > 0 ? printers.value[0].name : ''
      selectedPrinter.value = cfg.selected_printer || defaultPrinter
      port.value = cfg.port || 9999
      autoStart.value = cfg.auto_start || false
    }
    
    // Check if server is already running
    isRunning.value = await IsServerRunning()
    if (isRunning.value) {
      statusMessage.value = `Running on port ${port.value}`
      addActivity(`Server already running on port ${port.value}`, 'success')
    }
    
    // Listen for print events from backend (Wails v3 Events API)
    unsubPrintReceived = Events.On('print-received', (event) => {
      const data = event.data[0]
      lastPrintJob.value = data
      addActivity(`Print job received: ${data.type}`, 'print')
      showToast(`📄 Print job received (${data.type})`, 'info')
      statusMessage.value = `Print job received (${data.type})`
      setTimeout(() => {
        if (isRunning.value) {
          statusMessage.value = `Running on port ${port.value}`
        }
      }, 3000)
    })

    // Listen for print success/error events
    unsubPrintSuccess = Events.On('print-success', (event) => {
      const data = event.data[0]
      addActivity(`Print sent to ${data.printer}`, 'success')
      showToast(`✅ Print sent to ${data.printer}`, 'success')
      statusMessage.value = `✓ Print sent to ${data.printer}`
      setTimeout(() => {
        if (isRunning.value) {
          statusMessage.value = `Running on port ${port.value}`
        }
      }, 3000)
    })

    unsubPrintError = Events.On('print-error', (event) => {
      const data = event.data[0]
      addActivity(`Print failed: ${data.error}`, 'error')
      showToast(`❌ Print failed: ${data.error}`, 'error', 6000)
      statusMessage.value = `✗ Print failed: ${data.error}`
    })

    // App is ready with fade-in animation
    setTimeout(() => {
      isAppReady.value = true
      addActivity('Application ready', 'success')
    }, 100)

  } catch (error) {
    console.error('Failed to initialize:', error)
    statusMessage.value = 'Error: ' + error.message
    showToast(`❌ ${error.message}`, 'error')
  }
})

// Event unsubscribe functions
let unsubPrintReceived = null
let unsubPrintSuccess = null
let unsubPrintError = null

onUnmounted(() => {
  if (unsubPrintReceived) unsubPrintReceived()
  if (unsubPrintSuccess) unsubPrintSuccess()
  if (unsubPrintError) unsubPrintError()
})

// Actions
const handleSaveStart = async () => {
  isLoading.value = true
  try {
    if (isRunning.value) {
      // Stop server
      await StopServer()
      isRunning.value = false
      statusMessage.value = 'Stopped'
      addActivity('Server stopped', 'info')
      showToast('🛑 Server stopped', 'info')
    } else {
      // Save config and start server
      await SaveConfig(selectedPrinter.value, port.value, autoStart.value)
      await StartServer(port.value)
      isRunning.value = true
      statusMessage.value = `Running on port ${port.value}`
      addActivity(`Server started on port ${port.value}`, 'success')
      showToast(`🚀 Server started on port ${port.value}`, 'success')
    }
  } catch (error) {
    console.error('Error:', error)
    statusMessage.value = 'Error: ' + error.message
    addActivity(`Error: ${error.message}`, 'error')
    showToast(`❌ ${error.message}`, 'error')
  } finally {
    isLoading.value = false
  }
}

const handlePrintTest = async () => {
  isLoading.value = true
  try {
    // Save config first to ensure printer is selected
    await SaveConfig(selectedPrinter.value, port.value, autoStart.value)
    
    statusMessage.value = 'Printing test page...'
    addActivity('Printing test page...', 'print')
    await PrintTestPage()
    
    lastPrintJob.value = {
      type: 'test',
      content: 'Test Page',
      time: new Date().toISOString()
    }
    statusMessage.value = '✓ Test page sent!'
    addActivity('Test page sent successfully', 'success')
    showToast('✅ Test page sent to printer!', 'success')
    
    setTimeout(() => {
      statusMessage.value = isRunning.value ? `Running on port ${port.value}` : 'Stopped'
    }, 3000)
  } catch (error) {
    console.error('Print test error:', error)
    statusMessage.value = `✗ ${error.message || error}`
    addActivity(`Print test failed: ${error.message || error}`, 'error')
    showToast(`❌ ${error.message || error}`, 'error')
  } finally {
    isLoading.value = false
  }
}

const refreshPrinters = async () => {
  try {
    statusMessage.value = 'Refreshing printers...'
    addActivity('Refreshing printer list...', 'info')
    const printerList = await GetPrinters()
    printers.value = printerList || []
    statusMessage.value = isRunning.value ? `Running on port ${port.value}` : 'Stopped'
    addActivity(`Found ${printers.value.length} printer(s)`, 'success')
    showToast(`🖨️ Found ${printers.value.length} printer(s)`, 'info')
  } catch (error) {
    statusMessage.value = 'Error: ' + error.message
    showToast(`❌ ${error.message}`, 'error')
  }
}

// Window controls
const handleMinimize = () => {
  addActivity('Minimized to background', 'info')
  MinimizeToTray()
}

const handleQuit = () => {
  if (confirm('Are you sure you want to quit GoPrintBridge?')) {
    addActivity('Application closing...', 'info')
    QuitApp()
  }
}

// Format time for activity log
const formatTime = (date) => {
  return date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>

<template>
  <!-- Toast Container -->
  <div class="fixed top-4 right-4 z-50 space-y-2">
    <TransitionGroup name="toast">
      <div 
        v-for="toast in toasts" 
        :key="toast.id"
        class="glass px-4 py-3 rounded-xl shadow-lg max-w-xs transform transition-all duration-300"
        :class="{
          'border-green-400/30': toast.type === 'success',
          'border-red-400/30': toast.type === 'error',
          'border-mustard-400/30': toast.type === 'info',
          'opacity-0 translate-x-4': !toast.visible,
          'opacity-100 translate-x-0': toast.visible
        }"
      >
        <p class="text-sm text-white">{{ toast.message }}</p>
      </div>
    </TransitionGroup>
  </div>

  <!-- Main Background with Gradient -->
  <div class="min-h-screen w-full bg-gradient-to-br from-navy-900 via-navy-800 to-navy-700 flex items-center justify-center p-4">
    
    <!-- Glass Container with Animation -->
    <Transition name="fade-scale">
      <div v-if="isAppReady" class="glass-card w-full max-w-sm p-5 space-y-4 relative animate-fade-in">
        


        <!-- Header -->
        <div class="text-center space-y-1">
          <div class="flex items-center justify-center gap-2">
            <!-- Logo Icon -->
            <img 
              src="./assets/images/logo.png" 
              alt="GoPrintBridge Logo" 
              class="w-7 h-7 transition-transform duration-300 hover:scale-110"
            />
            <h1 class="text-xl font-bold text-white">GoPrintBridge</h1>
          </div>
          
          <!-- Status Indicator -->
          <div class="flex items-center justify-center gap-2">
            <span 
              class="status-dot w-2 h-2 rounded-full transition-colors duration-300"
              :class="isRunning ? 'bg-green-400 text-green-400' : 'bg-red-400 text-red-400'"
            ></span>
            <span class="text-xs text-white/70 transition-all duration-200">{{ statusMessage }}</span>
          </div>
        </div>

        <!-- Divider -->
        <div class="border-t border-white/10"></div>

        <!-- Form -->
        <div class="space-y-3">
          <!-- Printer Selection -->
          <div class="space-y-1.5 transition-opacity duration-200" :class="{ 'opacity-50': isLoading }">
            <div class="flex items-center justify-between">
              <label class="block text-xs font-medium text-white/80">Printer</label>
              <button 
                @click="refreshPrinters" 
                class="text-xs text-mustard-400 hover:text-mustard-300 flex items-center gap-1 transition-colors duration-200"
                :disabled="isLoading"
              >
                <svg class="w-3 h-3 transition-transform duration-300" :class="{ 'animate-spin': isLoading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                Refresh
              </button>
            </div>
            <select v-model="selectedPrinter" class="glass-select w-full transition-all duration-200" :disabled="isLoading">
              <option v-if="printers.length === 0" value="" class="bg-navy-900 text-white">
                No printers found
              </option>
              <option v-for="printer in printers" :key="printer.name" :value="printer.name" class="bg-navy-900 text-white">
                {{ printer.name }} ({{ printer.status }})
              </option>
            </select>
          </div>

          <!-- Port Input -->
          <div class="space-y-1.5 transition-opacity duration-200" :class="{ 'opacity-50': isRunning || isLoading }">
            <label class="block text-xs font-medium text-white/80 text-left">Port</label>
            <input 
              v-model.number="port"
              type="number" 
              class="glass-input w-full transition-all duration-200"
              placeholder="9999"
              min="1"
              max="65535"
              :disabled="isRunning || isLoading"
            />
          </div>

          <!-- Auto Start Toggle -->
          <div class="flex items-center justify-between transition-opacity duration-200" :class="{ 'opacity-50': isLoading }">
            <label class="text-xs font-medium text-white/80">Auto Start on Launch</label>
            <button 
              @click="autoStart = !autoStart"
              class="relative w-9 h-5 rounded-full transition-all duration-300"
              :class="autoStart ? 'bg-mustard-500' : 'bg-white/20'"
              :disabled="isLoading"
            >
              <span 
                class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-all duration-300 shadow-md"
                :class="autoStart ? 'translate-x-4' : 'translate-x-0'"
              ></span>
            </button>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="space-y-2 pt-1">
          <button 
            @click="handleSaveStart"
            class="w-full glass-btn-primary flex items-center justify-center gap-2 transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
            :disabled="isLoading"
          >
            <svg v-if="isLoading" class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <svg v-else-if="!isRunning" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                    d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
            </svg>
            {{ isLoading ? 'Processing...' : (isRunning ? 'Stop Server' : 'Save & Start') }}
          </button>
          
          <button 
            @click="handlePrintTest"
            class="w-full glass-btn flex items-center justify-center gap-2 transition-all duration-200 hover:scale-[1.02] active:scale-[0.98]"
            :disabled="isLoading"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            Print Test Page
          </button>
        </div>

        <!-- Activity Log -->
        <div v-if="activityLog.length > 0" class="space-y-1">
          <div class="border-t border-white/10 pt-2">
            <p class="text-xs text-white/40 mb-1">Recent Activity</p>
            <div class="space-y-1">
              <div 
                v-for="activity in activityLog"
                :key="activity.id"
                class="flex items-center gap-2 text-xs"
              >
                <span 
                  class="w-1.5 h-1.5 rounded-full flex-shrink-0"
                  :class="{
                    'bg-green-400': activity.type === 'success',
                    'bg-red-400': activity.type === 'error',
                    'bg-mustard-400': activity.type === 'print',
                    'bg-white/40': activity.type === 'info'
                  }"
                ></span>
                <span class="text-white/50 flex-shrink-0">{{ formatTime(activity.time) }}</span>
                <span class="text-white/70 truncate">{{ activity.message }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="text-center pt-1">
          <p class="text-xs text-white/40">GoPrintBridge v1.0.0</p>
        </div>
      </div>
    </Transition>

    <!-- Loading Skeleton -->
    <div v-if="!isAppReady" class="glass-card w-full max-w-sm p-5 space-y-4 animate-pulse">
      <div class="h-8 bg-white/10 rounded w-3/4 mx-auto"></div>
      <div class="h-4 bg-white/10 rounded w-1/2 mx-auto"></div>
      <div class="space-y-4">
        <div class="h-12 bg-white/10 rounded"></div>
        <div class="h-12 bg-white/10 rounded"></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Fade scale animation */
.fade-scale-enter-active,
.fade-scale-leave-active {
  transition: all 0.4s ease-out;
}
.fade-scale-enter-from,
.fade-scale-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

/* Toast animation */
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

/* List animation removed as per request */

/* Fade in animation */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
.animate-fade-in {
  animation: fadeIn 0.5s ease-out forwards;
}
</style>
