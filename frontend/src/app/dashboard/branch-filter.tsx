"use client";

import type { Branch } from "@/lib/types";

interface BranchFilterProps {
  branches: Branch[];
  currentBranchId: string;
}

export default function BranchFilter({
  branches,
  currentBranchId,
}: BranchFilterProps) {
  return (
    <div className="flex flex-col gap-2">
      <label className="text-[10px] font-black text-gray-400 uppercase tracking-widest ml-1">
        Lọc theo chi nhánh
      </label>
      <form>
        <select
          name="branch_id"
          defaultValue={currentBranchId}
          onChange={(e) => {
            const form = e.target.closest("form");
            if (form) form.submit();
          }}
          className="appearance-none rounded-2xl border-gray-100 text-sm font-bold focus:ring-primary-500 focus:border-primary-500 py-3 pl-4 pr-10 bg-white shadow-sm hover:shadow transition-all min-w-[240px] ring-1 ring-inset ring-gray-200"
        >
          <option value="">Tất cả chi nhánh</option>
          {branches.map((b) => (
            <option key={b.id} value={b.id}>
              {b.name}
            </option>
          ))}
        </select>
      </form>
    </div>
  );
}
