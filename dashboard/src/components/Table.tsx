import React, { useEffect } from "react";
import styled from "styled-components";
import {
  Column,
  Row,
  useGlobalFilter,
  usePagination,
  useTable,
} from "react-table";
import Loading from "components/Loading";
import Selector from "./Selector";

const GlobalFilter: React.FunctionComponent<any> = ({ setGlobalFilter }) => {
  const [value, setValue] = React.useState("");
  const onChange = (value: string) => {
    setValue(value);
    setGlobalFilter(value || undefined);
  };

  return (
    <SearchRow>
      <i className="material-icons">search</i>
      <SearchInput
        value={value}
        onChange={(e: any) => {
          onChange(e.target.value);
        }}
        placeholder="Search"
      />
    </SearchRow>
  );
};

export type TableProps = {
  columns: Column<any>[];
  data: any[];
  onRowClick?: (row: Row) => void;
  isLoading: boolean;
  disableGlobalFilter?: boolean;
  disableHover?: boolean;
  enablePagination?: boolean;
};

const MIN_PAGE_SIZE = 1;

const Table: React.FC<TableProps> = ({
  columns: columnsData,
  data,
  onRowClick,
  isLoading,
  disableGlobalFilter = false,
  disableHover,
  enablePagination,
}) => {
  const {
    getTableProps,
    getTableBodyProps,
    page,
    setGlobalFilter,
    prepareRow,
    headerGroups,
    visibleColumns,

    // Pagination options
    canPreviousPage,
    canNextPage,
    pageOptions,
    pageCount,
    gotoPage,
    nextPage,
    previousPage,
    setPageSize,
    state: { pageIndex, pageSize },
  } = useTable(
    {
      columns: columnsData,
      data,
    },
    useGlobalFilter,
    usePagination
  );

  useEffect(() => {
    if (!enablePagination) {
      setPageSize(data.length || MIN_PAGE_SIZE);
    }
  }, [data, enablePagination]);

  const renderRows = () => {
    if (isLoading) {
      return (
        <StyledTr disableHover={true} selected={false}>
          <StyledTd colSpan={visibleColumns.length}>
            <Loading />
          </StyledTd>
        </StyledTr>
      );
    }

    if (!page.length) {
      return (
        <StyledTr disableHover={true} selected={false}>
          <StyledTd colSpan={visibleColumns.length}>No data available</StyledTd>
        </StyledTr>
      );
    }
    return (
      <>
        {page.map((row) => {
          prepareRow(row);

          return (
            <StyledTr
              disableHover={disableHover}
              {...row.getRowProps()}
              enablePointer={!!onRowClick}
              onClick={() => onRowClick && onRowClick(row)}
              selected={false}
            >
              {/* TODO: This is actually broken, not sure why but we need the width to be properly setted, this is a temporary solution */}
              {row.cells.map((cell) => {
                return (
                  <StyledTd
                    {...cell.getCellProps()}
                    style={{
                      width: cell.column.totalWidth,
                    }}
                  >
                    {cell.render("Cell")}
                  </StyledTd>
                );
              })}
            </StyledTr>
          );
        })}
      </>
    );
  };

  return (
    <TableWrapper>
      {!disableGlobalFilter && (
        <GlobalFilter setGlobalFilter={setGlobalFilter} />
      )}
      <StyledTable {...getTableProps()}>
        <StyledTHead>
          {headerGroups.map((headerGroup) => (
            <StyledTr
              {...headerGroup.getHeaderGroupProps()}
              disableHover={true}
            >
              {headerGroup.headers.map((column) => (
                <StyledTh {...column.getHeaderProps()}>
                  {column.render("Header")}
                </StyledTh>
              ))}
            </StyledTr>
          ))}
        </StyledTHead>
        <tbody {...getTableBodyProps()}>{renderRows()}</tbody>
      </StyledTable>
      {enablePagination && (
        <FlexEnd style={{ marginTop: "15px" }}>
          <PageCountWrapper>
            Page size:
            <Selector
              activeValue={String(pageSize)}
              options={[
                {
                  label: "10",
                  value: "10",
                },
                {
                  label: "20",
                  value: "20",
                },
                {
                  label: "50",
                  value: "50",
                },
                {
                  label: "100",
                  value: "100",
                },
              ]}
              setActiveValue={(val) => setPageSize(Number(val))}
              width="70px"
            ></Selector>
          </PageCountWrapper>
          <PaginationActionsWrapper>
            <PaginationAction
              disabled={!canPreviousPage}
              onClick={previousPage}
            >
              {"<"}
            </PaginationAction>
            <PageCounter>
              {pageIndex + 1} of {pageCount}
            </PageCounter>
            <PaginationAction disabled={!canNextPage} onClick={nextPage}>
              {">"}
            </PaginationAction>
          </PaginationActionsWrapper>
        </FlexEnd>
      )}
    </TableWrapper>
  );
};

export default Table;

const TableWrapper = styled.div`
  padding-bottom: 20px;
`;

const FlexEnd = styled.div`
  display: flex;
  justify-content: flex-end;
  align-items: center;
  width: 100%;
`;

const PaginationActionsWrapper = styled.div``;

const PageCountWrapper = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-width: 160px;
  margin-right: 10px;
`;

const PaginationAction = styled.button`
  border: none;
  background: unset;
  color: white;
  padding: 10px;
  cursor: pointer;
  border-radius: 5px;
  :hover {
    background: #ffffff40;
  }

  :disabled {
    color: #ffffff88;
    cursor: unset;
    :hover {
      background: unset;
    }
  }
`;

const PageCounter = styled.span`
  margin: 0 5px;
`;

type StyledTrProps = {
  enablePointer?: boolean;
  disableHover?: boolean;
  selected?: boolean;
};

export const StyledTr = styled.tr`
  line-height: 2.2em;
  background: ${(props: StyledTrProps) => (props.selected ? "#ffffff11" : "")};
  :hover {
    background: ${(props: StyledTrProps) =>
      props.disableHover ? "" : "#ffffff22"};
  }
  cursor: ${(props: StyledTrProps) =>
    props.enablePointer ? "pointer" : "unset"};
`;

export const StyledTd = styled.td`
  font-size: 13px;
  color: #ffffff;
  :first-child {
    padding-left: 10px;
  }
  :last-child {
    padding-right: 10px;
  }
  user-select: text;
`;

export const StyledTHead = styled.thead`
  width: 100%;
  border-top: 1px solid #aaaabb22;
  border-bottom: 1px solid #aaaabb22;
  position: sticky;
`;

export const StyledTh = styled.th`
  text-align: left;
  font-size: 13px;
  font-weight: 500;
  color: #aaaabb;
  :first-child {
    padding-left: 10px;
  }
  :last-child {
    padding-right: 10px;
  }
`;

export const StyledTable = styled.table`
  width: 100%;
  min-width: 500px;
  border-collapse: collapse;
`;

const SearchInput = styled.input`
  outline: none;
  border: none;
  font-size: 13px;
  background: none;
  width: 100%;
  color: white;
  padding: 0;
  height: 20px;
`;

const SearchRow = styled.div`
  display: flex;
  width: 100%;
  font-size: 13px;
  color: #ffffff55;
  border-radius: 4px;
  user-select: none;
  align-items: center;
  padding: 10px 0px;
  min-width: 300px;
  max-width: min-content;
  background: #ffffff11;
  margin-bottom: 15px;
  margin-top: 0px;
  i {
    width: 18px;
    height: 18px;
    margin-left: 12px;
    margin-right: 12px;
    font-size: 20px;
  }
`;
