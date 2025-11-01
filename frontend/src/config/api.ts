// API Configuration
// Fill in these URLs with your actual API endpoints

export const API_ENDPOINTS = {
  // Agent Management
  GET_AGENTS: '', // GET - Fetch all agents
  TRIGGER_METRICS: '', // POST - Trigger metrics collection
  RESTART_AGENT: '', // POST - Restart specific agent
  UNINSTALL_AGENT: '', // POST - Uninstall specific agent
  
  // Task Management
  ASSIGN_TASK: '', // POST - Assign task to agent
};

// Helper function to check if APIs are configured
export const isApiConfigured = () => {
  return Object.values(API_ENDPOINTS).every(url => url !== '');
};
