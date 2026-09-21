/**
 * Every string the admin can say.
 *
 * This is the shape the other catalogs are typed against, so a key added here
 * and forgotten elsewhere is a build error rather than a word in the wrong
 * language on somebody's screen.
 */
export const en = {
  // Navigation and the shell.
  "nav.content": "Content",
  "nav.site": "Site",
  "nav.posts": "Posts",
  "nav.pages": "Pages",
  "nav.taxonomies": "Taxonomies",
  "nav.settings": "Settings",
  "nav.search": "Search",
  "nav.language": "Language",
  "shell.runtime": "{store} store · {runtime}",
  "shell.connecting": "connecting",

  // The listing.
  "list.new": "New",
  "list.search": "Search title, excerpt and body",
  "list.title": "Title",
  "list.terms": "Terms",
  "list.status": "Status",
  "list.published": "Published",
  "list.loading": "Loading",
  "list.empty": "Nothing matches these filters",
  "list.more": "Load more",
  "list.of": "{shown} of {total}",
  "list.inSection_one": "{count} in this section",
  "list.inSection_other": "{count} in this section",
  "list.open": "Open on the site",
  "list.failed": "Could not load content",
  "list.filterAny": "any",
  "list.filterIn": "In",
  "list.filterTerm": "Term",

  // Statuses, which are values rather than labels and so are translated here.
  "status.draft": "draft",
  "status.published": "published",
  "status.scheduled": "scheduled",
  "status.archived": "archived",

  // Problems the index reports.
  "problems.notIndexed_one": "{count} file could not be indexed",
  "problems.notIndexed_other": "{count} files could not be indexed",

  // Taxonomies.
  "taxonomies.title": "Taxonomies",
  "taxonomies.description": "Terms are derived from what your content carries.",
  "taxonomies.terms_one": "{count} term",
  "taxonomies.terms_other": "{count} terms",

  // The editor.
  "editor.back": "Back",
  "editor.titlePlaceholder": "Title",
  "editor.save": "Save",
  "editor.saving": "saving",
  "editor.saved": "saved",
  "editor.unsaved": "unsaved",
  "editor.uploading": "uploading {count}",
  "editor.loading": "Loading",
  "editor.nothingToEdit": "Nothing to edit",
  "editor.slug": "Slug",
  "editor.slugPlaceholder": "derived from the title",
  "editor.publishedAt": "Published",
  "editor.commaSeparated": "comma separated",
  "editor.delete": "Delete",
  "editor.confirmDelete": "Delete this item and everything in its folder?",
  "editor.previewFailed": "The preview failed ({status})",

  // The conflict view.
  "conflict.title": "This changed while you were editing",
  "conflict.description":
    "Something else saved this item since you opened it. Nothing has been overwritten. Choose which version to keep.",
  "conflict.ours": "Yours",
  "conflict.oursNote": "what you have been writing",
  "conflict.theirs": "Stored",
  "conflict.theirsNote": "what is on disk now",
  "conflict.unreadable": "(could not be read)",
  "conflict.keepEditing": "Keep editing",
  "conflict.takeTheirs": "Discard mine, load stored",
  "conflict.keepOurs": "Overwrite with mine",

  // Publishing.
  "publish.title": "Publish",
  "publish.delivery": "Delivery",
  "publish.action": "Publish",
  "publish.anyway": "Publish anyway",
  "publish.working": "Publishing",
  "publish.saved": "Saved",
  "publish.committed": "Committed",
  "publish.pushed": "Pushed",
  "publish.deployed": "Deployed",
  "publish.deployedNote": "your host reports this",
  "publish.uncommitted_one": "{count} file uncommitted",
  "publish.uncommitted_other": "{count} files uncommitted",
  "publish.toPush_one": "{count} commit to push",
  "publish.toPush_other": "{count} commits to push",
  "publish.failed": "The publish did not finish",

  // Settings.
  "settings.title": "Settings",
  "settings.description": "Written to kite.yaml, leaving everything else in it alone.",
  "settings.site": "Site",
  "settings.theme": "Theme",
  "settings.themeNote": "{theme} declares these itself.",
  "settings.themeEmpty": "This theme declares no settings.",
  "settings.saving": "Saving",
  "settings.save": "Save",
  "settings.loadFailed": "Could not load the settings",
  "settings.siteTitle": "Title",
  "settings.siteDescription": "Description",
  "settings.siteBaseURL": "Base URL",
  "settings.siteLanguage": "Language",

  // Server problem codes. Every one the API can return is stable, so it is
  // translated here rather than shown in whatever language the server speaks.
  "problem.not_found": "That does not exist.",
  "problem.invalid_request": "Something in that request was not understood.",
  "problem.unsupported_query": "That filter or ordering is not supported.",
  "problem.conflict": "This item changed since it was loaded.",
  "problem.read_only": "This server does not accept changes.",
  "problem.internal": "Something went wrong on the server.",
  "problem.publish_refused": "The publish was refused.",
  "problem.publish_needs_confirmation": "Read the warnings, then publish again.",
  "problem.publish_failed": "The publish did not finish.",
  "problem.not_a_repository": "This project is not a git repository.",
  "problem.detached_head": "HEAD is not on a branch.",
  "problem.operation_in_flight": "A git operation is already in progress.",
  "problem.submodule": "This repository is a submodule of another one.",
  "problem.lfs_missing": "Some of these files need git-lfs, which is not installed.",
  "problem.nothing_to_publish": "There is nothing to publish.",
  "problem.no_remote": "This repository has no remote.",
  "problem.no_credentials": "Git could not authenticate without asking.",
  "problem.remote_moved": "The remote has commits this branch does not.",
  "problem.staged_elsewhere": "You have already staged changes to these files.",
  "problem.path_not_changed": "Nothing changed in those paths.",
  "problem.quota_exceeded": "A file is larger than the host will accept.",
  "problem.locked": "Another publish is in progress.",
  "problem.git_missing": "Git is not installed.",
  "problem.git_failed": "Git refused the command.",
} as const;
