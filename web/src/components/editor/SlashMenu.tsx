import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { Extension, type Editor, type Range } from "@tiptap/core";
import { PluginKey } from "@tiptap/pm/state";
import { ReactRenderer } from "@tiptap/react";
import { Suggestion, type SuggestionKeyDownProps, type SuggestionProps } from "@tiptap/suggestion";
import type { LucideIcon } from "lucide-react";
import { cn } from "cn";

/** One thing a "/" can insert. */
export interface SlashItem {
  id: string;
  label: string;
  hint?: string;
  /** Extra words a search matches, in whatever languages the author may type. */
  keywords?: string;
  icon: LucideIcon;
  run: (editor: Editor, range: Range) => void;
}

interface Options {
  /** items is read each time the menu opens, so labels follow the language. */
  items: () => SlashItem[];
  empty: () => string;
}

function matches(item: SlashItem, query: string): boolean {
  const q = query.toLowerCase();
  return `${item.label} ${item.hint ?? ""} ${item.keywords ?? ""}`.toLowerCase().includes(q);
}

/**
 * A menu of blocks, opened by typing "/" at the start of a line or after a
 * space. The list is rendered with React and positioned by the suggestion
 * plugin, which also closes it on Escape.
 */
export const Slash = Extension.create<Options>({
  name: "slash",

  addOptions() {
    return { items: () => [], empty: () => "" };
  },

  addProseMirrorPlugins() {
    const options = this.options;
    return [
      Suggestion<SlashItem, SlashItem>({
        editor: this.editor,
        pluginKey: new PluginKey("slash"),
        char: "/",
        allowSpaces: false,
        startOfLine: false,
        shouldShow: ({ editor }) => !editor.isActive("codeBlock"),
        items: ({ query }) => options.items().filter((item) => matches(item, query)),
        command: ({ editor, range, props }) => props.run(editor, range),
        render: () => {
          let renderer: ReactRenderer<ListHandle, ListProps> | null = null;
          let unmount: (() => void) | null = null;

          return {
            onStart: (props) => {
              renderer = new ReactRenderer(SlashList, {
                props: { ...props, empty: options.empty() },
                editor: props.editor,
              });
              unmount = props.mount(renderer.element);
            },
            onUpdate: (props) => {
              renderer?.updateProps({ ...props, empty: options.empty() });
            },
            onKeyDown: (props) => renderer?.ref?.onKeyDown(props) ?? false,
            onExit: () => {
              unmount?.();
              renderer?.destroy();
              renderer = null;
              unmount = null;
            },
          };
        },
      }),
    ];
  },
});

interface ListProps extends SuggestionProps<SlashItem, SlashItem> {
  empty: string;
}

interface ListHandle {
  onKeyDown: (props: SuggestionKeyDownProps) => boolean;
}

const SlashList = forwardRef<ListHandle, ListProps>(function SlashList(
  { items, command, empty },
  ref,
) {
  const [index, setIndex] = useState(0);
  const list = useRef<HTMLDivElement>(null);

  useEffect(() => setIndex(0), [items]);
  useEffect(() => {
    list.current
      ?.querySelector("[data-selected=true]")
      ?.scrollIntoView({ block: "nearest" });
  }, [index]);

  useImperativeHandle(
    ref,
    () => ({
      onKeyDown: ({ event }) => {
        if (items.length === 0) return false;
        if (event.key === "ArrowUp") {
          setIndex((i) => (i + items.length - 1) % items.length);
          return true;
        }
        if (event.key === "ArrowDown") {
          setIndex((i) => (i + 1) % items.length);
          return true;
        }
        if (event.key === "Enter") {
          const item = items[index];
          if (item) command(item);
          return true;
        }
        return false;
      },
    }),
    [items, index, command],
  );

  return (
    <div
      ref={list}
      role="listbox"
      className="z-50 max-h-80 w-64 overflow-y-auto rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10"
    >
      {items.length === 0 ? (
        <div className="px-2 py-5 text-center text-sm text-muted-foreground">{empty}</div>
      ) : (
        items.map((item, i) => (
          <button
            key={item.id}
            type="button"
            role="option"
            aria-selected={i === index}
            data-selected={i === index}
            className={cn(
              "flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-sm outline-none",
              "data-[selected=true]:bg-muted",
            )}
            onMouseEnter={() => setIndex(i)}
            // Keeps the selection the command is about to act on.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => command(item)}
          >
            <span className="flex size-7 shrink-0 items-center justify-center rounded-md border bg-background text-muted-foreground">
              <item.icon className="size-4" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate">{item.label}</span>
              {item.hint && (
                <span className="block truncate text-xs text-muted-foreground">{item.hint}</span>
              )}
            </span>
          </button>
        ))
      )}
    </div>
  );
});
