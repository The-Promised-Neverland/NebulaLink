import { useState, useMemo, useCallback, useEffect, memo } from "react";
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
  const [hasInitialized, setHasInitialized] = useState(false);

  // Build tree structure from flat file list
  const treeData = useMemo(() => {
    if (!snapshot?.directory?.files) return [];

    const root: FileTreeNode[] = [];
    const nodeMap = new Map<string, FileTreeNode>();

    // Helper to get or create a directory node
    const getOrCreateDir = (dirPath: string, dirName: string): FileTreeNode => {
      if (nodeMap.has(dirPath)) {
        return nodeMap.get(dirPath)!;
      }

      const dirNode: FileTreeNode = {
        name: dirName,
        path: dirPath,
        size: 0,
        modified: new Date().toISOString(),
        type: "directory",
        children: [],
      };
      nodeMap.set(dirPath, dirNode);
      return dirNode;
    };

    // Process all files
    snapshot.directory.files.forEach((file) => {
      const normalizedPath = file.path.replace(/\/$/, "").replace(/\\/g, "/");
      if (!normalizedPath || normalizedPath === "." || normalizedPath === "..") return;

      const pathParts = normalizedPath.split("/").filter(p => p);
      if (pathParts.length === 0) return;

      const node: FileTreeNode = {
        name: file.name,
        path: normalizedPath,
        size: file.size,
        modified: file.modified,
        type: file.type,
        children: file.type === "directory" ? [] : undefined,
      };

      if (pathParts.length === 1) {
        // Root level
        if (file.type === "directory") {
          nodeMap.set(normalizedPath, node);
        }
        root.push(node);
      } else {
        // Nested - build parent chain
        let currentPath = "";
        let parent: FileTreeNode | null = null;

        for (let i = 0; i < pathParts.length - 1; i++) {
          const part = pathParts[i];
          currentPath = currentPath ? `${currentPath}/${part}` : part;
          
          const dirNode = getOrCreateDir(currentPath, part);
          if (i === 0 && !root.includes(dirNode)) {
            root.push(dirNode);
          }
          if (parent && parent.children && !parent.children.includes(dirNode)) {
            parent.children.push(dirNode);
          }
          parent = dirNode;
        }

        // Add the file/directory to its parent
        if (parent && parent.children) {
          parent.children.push(node);
          if (file.type === "directory") {
            nodeMap.set(normalizedPath, node);
          }
        } else {
          // Fallback: add to root
          root.push(node);
        }
      }
    });

    // Sort recursively - directories first, then files
    const sortTree = (nodes: FileTreeNode[]): void => {
      // Sort root level first
      nodes.sort((a, b) => {
        if (a.type !== b.type) return a.type === "directory" ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
      
      // Sort children recursively
      nodes.forEach(node => {
        if (node.children) {
          node.children.sort((a, b) => {
            if (a.type !== b.type) return a.type === "directory" ? -1 : 1;
            return a.name.localeCompare(b.name);
          });
          sortTree(node.children);
        }
      });
    };

    sortTree(root);

    return root;
  }, [snapshot]);

  // Auto-expand root directories on first load only (not when user collapses)
  useEffect(() => {
    if (snapshot?.directory?.files && !hasInitialized && treeData.length > 0) {
      const rootDirs = treeData
        .filter(n => n.type === "directory" && n.children && n.children.length > 0)
        .map(n => n.path);
      if (rootDirs.length > 0) {
        setExpandedPaths(new Set(rootDirs));
        setHasInitialized(true);
      }
    }
  }, [snapshot, treeData, hasInitialized]);

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
      // Force a new Set instance to trigger re-render
      return new Set(next);
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

  const handleToggle = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (isDirectory && hasChildren) {
      onToggle(node.path);
    }
  }, [isDirectory, hasChildren, node.path, onToggle]);

  return (
    <div>
      <div
        className={cn(
          "file-tree-item flex items-center gap-2 px-2 py-1.5 rounded-md transition-colors",
          isDirectory && hasChildren && "cursor-pointer hover:bg-muted/70",
          !hasChildren && isDirectory && "opacity-60",
          !isDirectory && "hover:bg-muted/30"
        )}
        style={{ paddingLeft: `${level * 20 + 8}px` }}
        onClick={handleToggle}
      >
        {/* Expand/collapse icon */}
        <div className="w-4 flex-shrink-0 flex items-center justify-center">
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )
          ) : isDirectory ? (
            <div className="w-1 h-1 rounded-full bg-muted-foreground/30" />
          ) : (
            <div className="w-4" />
          )}
        </div>

        {/* File/folder icon */}
        {isDirectory ? (
          <Folder className={cn("h-4 w-4 flex-shrink-0", isExpanded && hasChildren ? "text-primary" : "text-muted-foreground")} />
        ) : (
          <FileText className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
        )}

        {/* Name */}
        <span className={cn("flex-1 text-sm truncate", isDirectory ? "font-medium" : "font-normal")}>
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

      {/* Children - render when expanded */}
      {isExpanded && hasChildren && (
        <div>
          {node.children!.map((child) => (
            <TreeNode
              key={`${child.path}-${expandedPaths.has(child.path)}`}
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
