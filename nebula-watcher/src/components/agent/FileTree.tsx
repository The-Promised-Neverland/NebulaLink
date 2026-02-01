import { useState, useMemo, useCallback } from "react";
import {
  Folder,
  File,
  ChevronRight,
  ChevronDown,
  Search,
  HardDrive,
  FileText,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import type { FileInfo, DirectorySnapshot, FileTreeNode } from "@/types";

interface FileTreeProps {
  snapshot: DirectorySnapshot | undefined;
}

export function FileTree({ snapshot }: FileTreeProps) {
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");

  // Build tree structure from flat file list
  const treeData = useMemo(() => {
    if (!snapshot?.directory?.files) return [];

    const root: FileTreeNode[] = [];
    const pathMap = new Map<string, FileTreeNode>();

    // Sort files: directories first, then by name
    const sortedFiles = [...snapshot.directory.files].sort((a, b) => {
      if (a.type !== b.type) return a.type === "directory" ? -1 : 1;
      return a.name.localeCompare(b.name);
    });

    sortedFiles.forEach((file) => {
      const pathParts = file.path.replace(/\/$/, "").split("/");
      const node: FileTreeNode = {
        name: file.name,
        path: file.path,
        size: file.size,
        modified: file.modified,
        type: file.type,
        children: file.type === "directory" ? [] : undefined,
      };

      if (pathParts.length === 1) {
        // Root level item
        root.push(node);
        if (file.type === "directory") {
          pathMap.set(file.path.replace(/\/$/, ""), node);
        }
      } else {
        // Nested item - find parent
        const parentPath = pathParts.slice(0, -1).join("/");
        const parent = pathMap.get(parentPath);
        
        if (parent && parent.children) {
          parent.children.push(node);
          if (file.type === "directory") {
            pathMap.set(file.path.replace(/\/$/, ""), node);
          }
        } else {
          // Parent not found, add to root (shouldn't happen with proper data)
          root.push(node);
        }
      }
    });

    return root;
  }, [snapshot]);

  // Filter tree based on search
  const filteredTree = useMemo(() => {
    if (!searchQuery.trim()) return treeData;

    const query = searchQuery.toLowerCase();
    
    const filterNode = (node: FileTreeNode): FileTreeNode | null => {
      const matches = node.name.toLowerCase().includes(query);
      
      if (node.type === "directory" && node.children) {
        const filteredChildren = node.children
          .map(filterNode)
          .filter((n): n is FileTreeNode => n !== null);
        
        if (matches || filteredChildren.length > 0) {
          return { ...node, children: filteredChildren };
        }
        return null;
      }
      
      return matches ? node : null;
    };

    return treeData
      .map(filterNode)
      .filter((n): n is FileTreeNode => n !== null);
  }, [treeData, searchQuery]);

  const toggleExpand = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  const formatSize = (bytes: number): string => {
    if (bytes === 0) return "-";
    const units = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
  };

  const formatDate = (isoString: string): string => {
    const date = new Date(isoString);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  if (!snapshot) {
    return (
      <div className="glass-card rounded-xl p-8 text-center">
        <HardDrive className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <h3 className="text-lg font-medium text-foreground mb-2">No Directory Data</h3>
        <p className="text-sm text-muted-foreground">
          Waiting for directory snapshot from agent...
        </p>
      </div>
    );
  }

  return (
    <div className="glass-card rounded-xl overflow-hidden">
      {/* Header */}
      <div className="border-b border-border p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <Folder className="h-5 w-5 text-primary" />
            <h3 className="font-semibold text-foreground">File Browser</h3>
          </div>
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <span>
              <span className="text-foreground font-mono">{snapshot.directory.total_files}</span> files
            </span>
            <span>
              <span className="text-foreground font-mono">{formatSize(snapshot.directory.total_size)}</span> total
            </span>
          </div>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search files..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 bg-muted/50 border-border"
          />
        </div>
      </div>

      {/* Tree content */}
      <div className="max-h-[400px] overflow-auto p-2">
        {filteredTree.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {searchQuery ? "No matching files found" : "No files available"}
          </div>
        ) : (
          <div className="space-y-0.5">
            {filteredTree.map((node) => (
              <TreeNode
                key={node.path}
                node={node}
                level={0}
                expandedPaths={expandedPaths}
                onToggle={toggleExpand}
                formatSize={formatSize}
                formatDate={formatDate}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

interface TreeNodeProps {
  node: FileTreeNode;
  level: number;
  expandedPaths: Set<string>;
  onToggle: (path: string) => void;
  formatSize: (bytes: number) => string;
  formatDate: (iso: string) => string;
}

function TreeNode({
  node,
  level,
  expandedPaths,
  onToggle,
  formatSize,
  formatDate,
}: TreeNodeProps) {
  const isExpanded = expandedPaths.has(node.path);
  const isDirectory = node.type === "directory";
  const hasChildren = isDirectory && node.children && node.children.length > 0;

  return (
    <div>
      <div
        className={cn(
          "file-tree-item flex items-center gap-2 px-2 py-1.5 rounded-md",
          isDirectory && "cursor-pointer",
          "hover:bg-muted/50"
        )}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={() => isDirectory && onToggle(node.path)}
      >
        {/* Expand/collapse icon */}
        <div className="w-4 flex-shrink-0">
          {hasChildren && (
            isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )
          )}
        </div>

        {/* File/folder icon */}
        {isDirectory ? (
          <Folder className={cn("h-4 w-4 flex-shrink-0", isExpanded ? "text-primary" : "text-muted-foreground")} />
        ) : (
          <FileText className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
        )}

        {/* Name */}
        <span className={cn("flex-1 text-sm truncate", isDirectory ? "font-medium" : "")}>
          {node.name}
        </span>

        {/* Size */}
        <span className="text-xs font-mono text-muted-foreground w-20 text-right">
          {formatSize(node.size)}
        </span>

        {/* Modified */}
        <span className="text-xs text-muted-foreground w-32 text-right hidden md:block">
          {formatDate(node.modified)}
        </span>
      </div>

      {/* Children */}
      {isExpanded && hasChildren && (
        <div className="animate-fade-in">
          {node.children!.map((child) => (
            <TreeNode
              key={child.path}
              node={child}
              level={level + 1}
              expandedPaths={expandedPaths}
              onToggle={onToggle}
              formatSize={formatSize}
              formatDate={formatDate}
            />
          ))}
        </div>
      )}
    </div>
  );
}
