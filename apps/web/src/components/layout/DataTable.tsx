import type { ReactNode } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../ui/table";
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from "../ui/pagination";
import { Skeleton } from "../ui/skeleton";
import { EmptyState } from "./EmptyState";
import * as m from "../../paraglide/messages";

export interface DataTableColumn<T> {
  id: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  className?: string;
  /** 传入即视为可排序列，点击表头触发 onSortChange */
  sortable?: boolean;
}

export interface DataTablePagination {
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
}

export interface DataTableSort {
  columnId: string;
  direction: "asc" | "desc";
}

export interface DataTableProps<T> {
  columns: Array<DataTableColumn<T>>;
  data: Array<T>;
  getRowId: (row: T) => string;
  isLoading?: boolean;
  emptyState?: ReactNode;
  pagination?: DataTablePagination;
  sort?: DataTableSort;
  onSortChange?: (columnId: string) => void;
  skeletonRows?: number;
}

/**
 * `ui/table` 的薄封装：内置分页与排序表头，加载态骨架屏、空态统一交给
 * `EmptyState`（DESIGN.md §3/§8 的三态基线：空/加载/错误——错误态由调用方
 * 结合 `AppErrorBoundary`/`lib/errors.ts` 处理，本组件只负责空与加载）。
 */
export function DataTable<T>({
  columns,
  data,
  getRowId,
  isLoading = false,
  emptyState,
  pagination,
  sort,
  onSortChange,
  skeletonRows = 5,
}: DataTableProps<T>) {
  return (
    <div className="space-y-3">
      <div className="rounded-lg border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              {columns.map((column) => (
                <TableHead
                  key={column.id}
                  className={column.className}
                  aria-sort={
                    sort?.columnId === column.id
                      ? sort.direction === "asc"
                        ? "ascending"
                        : "descending"
                      : undefined
                  }
                >
                  {column.sortable ? (
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 hover:text-foreground"
                      onClick={() => onSortChange?.(column.id)}
                    >
                      {column.header}
                    </button>
                  ) : (
                    column.header
                  )}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading &&
              Array.from({ length: skeletonRows }).map((_, index) => (
                <TableRow key={`skeleton-${index}`}>
                  {columns.map((column) => (
                    <TableCell key={column.id}>
                      <Skeleton className="h-4 w-full max-w-[10rem]" />
                    </TableCell>
                  ))}
                </TableRow>
              ))}

            {!isLoading &&
              data.map((row) => (
                <TableRow key={getRowId(row)}>
                  {columns.map((column) => (
                    <TableCell key={column.id} className={column.className}>
                      {column.cell(row)}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
          </TableBody>
        </Table>

        {!isLoading && data.length === 0 && (emptyState ?? (
          <EmptyState
            title={m.data_table_empty()}
            className="rounded-none border-0 border-t border-border"
          />
        ))}
      </div>

      {pagination && pagination.pageCount > 1 && (
        <Pagination className="justify-between">
          <span className="font-mono text-xs text-muted-foreground">
            {m.data_table_page_of({
              current: pagination.page,
              total: pagination.pageCount,
            })}
          </span>
          <PaginationContent>
            <PaginationItem>
              <PaginationPrevious
                disabled={pagination.page <= 1}
                onClick={() => pagination.onPageChange(pagination.page - 1)}
              />
            </PaginationItem>
            <PaginationItem>
              <PaginationNext
                disabled={pagination.page >= pagination.pageCount}
                onClick={() => pagination.onPageChange(pagination.page + 1)}
              />
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      )}
    </div>
  );
}
