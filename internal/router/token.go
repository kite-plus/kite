package router

// All /tokens endpoints (list / create / delete) moved to the typed
// internal/api package and are registered via api.Register at engine setup
// time. This file is kept (instead of deleted) so the symbol naming
// conventions for new domain registrations stay obvious — when more typed
// migrations land, the empty stubs serve as breadcrumbs to the new home.
