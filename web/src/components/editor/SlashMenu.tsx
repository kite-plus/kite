import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type ComponentType } from "react";
import { Extension, type Editor, type Range } from "@tiptap/core";
import { PluginKey } from "@tiptap/pm/state";
import { ReactRenderer } from "@tiptap/react";
import { Suggestion, type SuggestionKeyDownProps, type SuggestionProps } from "@tiptap/suggestion";

import { Button } from "@/components/tiptap-ui-primitive/button";
import { Card, CardBody, CardGroupLabel, CardItemGroup } from "@/components/tiptap-ui-primitive/card";

/** One thing a "/" can insert. */
export interface SlashItem {
  id: string;
  label: string;
  /** The heading the item is listed under; items sharing one are listed together. */
  group: string;
  /** Extra words a search matches, in whatever languages the author may type. */
  keywords?: string;
  icon: ComponentType<{ className?: string }>;
  run: (editor: Editor, range: Range) => void;
}

interface Options {
  /** items is read each time the menu opens, so labels follow the language. */
  items: () => SlashItem[];
  empty: () => string;
}

function matches(item: SlashItem, query: string): boolean {
  const q = query.toLowerCase();
  return `${item.label} ${item.keywords ?? ""}`.toLowerCase().includes(q);
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
                className: "kite-slash-popup",
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
      ?.querySelector("[data-highlighted=true]")
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

  // Items arrive group by group, so the keyboard walks them as they are drawn.
  const groups = new Map<string, { item: SlashItem; at: number }[]>();
  items.forEach((item, at) => {
    const group = groups.get(item.group) ?? [];
    group.push({ item, at });
    groups.set(item.group, group);
  });

  return (
    <Card ref={list} role="listbox" className="kite-slash">
      <CardBody>
        {items.length === 0 ? (
          <div className="kite-slash-empty">{empty}</div>
        ) : (
          [...groups].map(([label, entries]) => (
            <CardItemGroup key={label}>
              <CardGroupLabel>{label}</CardGroupLabel>
              {entries.map(({ item, at }) => (
                <Button
                  key={item.id}
                  type="button"
                  variant="ghost"
                  role="option"
                  aria-selected={at === index}
                  data-highlighted={at === index}
                  onMouseEnter={() => setIndex(at)}
                  // Keeps the selection the command is about to act on.
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={() => command(item)}
                >
                  <item.icon className="tiptap-button-icon" />
                  <span className="tiptap-button-text">{item.label}</span>
                </Button>
              ))}
            </CardItemGroup>
          ))
        )}
      </CardBody>
    </Card>
  );
});
